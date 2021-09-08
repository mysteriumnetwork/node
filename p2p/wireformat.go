/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strconv"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/node/p2p/compat"
	"github.com/mysteriumnetwork/node/pb"
)

const maxTransportMsgLen = 128 * 1024

func init() {
	// This is needed to initialize global common headers map state internally
	// before reading header values so race detector doesn't fail unit tests
	// when multiple goroutines reads conn data headers.
	textproto.CanonicalMIMEHeaderKey("")
}

type wireReader interface {
	readMsg(*transportMsg) error
}

type wireWriter interface {
	writeMsg(*transportMsg) error
}

const (
	headerFieldRequestID = "Request-ID"
	headerFieldTopic     = "Topic"
	headerStatusCode     = "Status-Code"
	headerMsg            = "Message"
)

func newCompatibleWireReader(c io.Reader, peerCompatibility int) wireReader {
	if compat.FeaturePBP2P(peerCompatibility) {
		log.Debug().Msg("Using protobufWireReader")
		return newProtobufWireReader(c)
	}
	log.Debug().Msg("Using textWireReader")
	return newTextWireReader(c)
}

func newCompatibleWireWriter(c io.Writer, peerCompatibility int) wireWriter {
	if compat.FeaturePBP2P(peerCompatibility) {
		log.Debug().Msg("Using protobufWireWriter")
		return newProtobufWireWriter(c)
	}
	log.Debug().Msg("Using textWireWriter")
	return newTextWireWriter(c)
}

type textWireReader textproto.Reader

type textWireWriter textproto.Writer

func newTextWireReader(c io.Reader) *textWireReader {
	return (*textWireReader)(textproto.NewReader(bufio.NewReader(c)))
}

func (r *textWireReader) readMsg(m *transportMsg) error {
	// Read header.
	header, err := (*textproto.Reader)(r).ReadMIMEHeader()
	if err != nil {
		return fmt.Errorf("could not read mime header: %w", err)
	}
	id, err := strconv.ParseUint(header.Get(headerFieldRequestID), 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse request id: %w", err)
	}
	m.id = id
	statusCode, err := strconv.ParseUint(header.Get(headerStatusCode), 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse status code: %w", err)
	}
	m.statusCode = statusCode
	m.topic = header.Get(headerFieldTopic)
	m.msg = header.Get(headerMsg)

	// Read data.
	data, err := (*textproto.Reader)(r).ReadDotBytes()
	if err != nil {
		return fmt.Errorf("could not read dot bytes: %w", err)
	}
	if len(data) > 0 {
		m.data = data[:len(data)-1]
	}
	return nil
}

func newTextWireWriter(c io.Writer) *textWireWriter {
	return (*textWireWriter)(textproto.NewWriter(bufio.NewWriter(c)))
}

func (w *textWireWriter) writeMsg(m *transportMsg) error {
	dotWriter := (*textproto.Writer)(w).DotWriter()
	var header bytes.Buffer
	header.WriteString(fmt.Sprintf("%s:%d\r\n", headerFieldRequestID, m.id))
	header.WriteString(fmt.Sprintf("%s:%s\r\n", headerFieldTopic, m.topic))
	header.WriteString(fmt.Sprintf("%s:%d\r\n", headerStatusCode, m.statusCode))
	header.WriteString(fmt.Sprintf("%s:%s\r\n", headerMsg, m.msg))
	header.WriteByte('\n')
	dotWriter.Write(header.Bytes())
	dotWriter.Write(m.data)
	return dotWriter.Close()
}

type protobufWireReader struct {
	r      *bufio.Reader
	closed bool
}

func newProtobufWireReader(c io.Reader) *protobufWireReader {
	return &protobufWireReader{
		r: bufio.NewReader(c),
	}
}

func (r *protobufWireReader) readMsg(m *transportMsg) error {
	if r.closed {
		return io.EOF
	}

	msgLen, err := binary.ReadUvarint(r.r)
	if err != nil {
		r.closed = true
		return err
	}

	if msgLen > maxTransportMsgLen {
		r.closed = true
		return io.EOF
	}

	msgBytes := make([]byte, msgLen)
	_, err = io.ReadFull(r.r, msgBytes)
	if err != nil {
		r.closed = true
		return err
	}

	var pbMsg pb.P2PChannelEnvelope
	err = proto.Unmarshal(msgBytes, &pbMsg)
	if err != nil {
		return err
	}

	m.id = pbMsg.ID
	m.statusCode = pbMsg.StatusCode
	m.topic = pbMsg.Topic
	m.msg = pbMsg.Msg
	m.data = pbMsg.Data

	return nil
}

type protobufWireWriter struct {
	w *bufio.Writer
}

func newProtobufWireWriter(c io.Writer) *protobufWireWriter {
	return &protobufWireWriter{
		w: bufio.NewWriter(c),
	}
}

func (w *protobufWireWriter) writeMsg(m *transportMsg) error {
	pbMsg := pb.P2PChannelEnvelope{
		ID:         m.id,
		StatusCode: m.statusCode,
		Topic:      m.topic,
		Msg:        m.msg,
		Data:       m.data,
	}

	msgBytes, err := proto.Marshal(&pbMsg)
	if err != nil {
		return err
	}

	msgLen := len(msgBytes)
	if msgLen > maxTransportMsgLen {
		return errors.New("can't marshal: message too long")
	}

	lenBuf := make([]byte, binary.MaxVarintLen64)
	lenBufLen := binary.PutUvarint(lenBuf, uint64(msgLen))
	lenBuf = lenBuf[:lenBufLen]

	_, err = w.w.Write(lenBuf)
	if err != nil {
		return err
	}

	_, err = w.w.Write(msgBytes)
	if err != nil {
		return err
	}

	err = w.w.Flush()
	if err != nil {
		return err
	}

	return nil
}
