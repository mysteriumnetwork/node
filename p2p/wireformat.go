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
	"fmt"
	"bytes"
	"strconv"
	"io"
	"net/textproto"
)

func init() {
	// This is needed to initialise global common headers map state internally
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
