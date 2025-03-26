/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services/quic/streams"
	"github.com/mysteriumnetwork/node/trace"
)

type channelQuic struct {
	conn     *streams.QuicConnection
	tr       *streams.QuicConnection
	handlers map[string]HandlerFunc
	id       identity.Identity
	release  func()

	compatibility int

	mu     sync.Mutex
	tracer *trace.Tracer
}

// ServiceConn represents service connection.
type ServiceConn interface {
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

func newChannelQuic(c ServiceConn, id identity.Identity, peerCompatibility int) *channelQuic {
	return &channelQuic{
		conn:          c.(*streams.QuicConnection),
		id:            id,
		compatibility: peerCompatibility,
	}
}

func (c *channelQuic) Handle(topic string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlers == nil {
		c.handlers = make(map[string]HandlerFunc)
	}

	c.handlers[topic] = handler
}

func (c *channelQuic) ID() string {
	return c.id.Address
}

func (c *channelQuic) Conn() ServiceConn {
	return c.conn
}

func (c *channelQuic) Tracer() *trace.Tracer {
	return c.tracer
}

func (c *channelQuic) ServiceConn() ServiceConn {
	return c.tr
}

func (c *channelQuic) Send(ctx context.Context, topic string, msg *Message) (*Message, error) {
	s, err := c.conn.OpenStream()
	if err != nil {
		return nil, err
	}

	defer func() {
		s.CancelWrite(0)
	}()

	log.Debug().Str("topic", topic).Msg("Opened QUIC stream to send a message")

	sendMsg := transportMsg{
		id:    uint64(s.StreamID()),
		topic: topic,
		data:  msg.Data,
	}
	if err := sendMsg.writeTo(newCompatibleWireWriter(s, c.compatibility)); err != nil {
		return nil, err
	}

	readMsg := transportMsg{}
	if err := readMsg.readFrom(newCompatibleWireReader(s, c.compatibility)); err != nil {
		return nil, err
	}

	if readMsg.statusCode != statusCodeOK {
		if readMsg.statusCode == statusCodePublicErr {
			return nil, fmt.Errorf("public peer error: %s", string(readMsg.data))
		}
		if readMsg.statusCode == statusCodeHandlerNotFoundErr {
			return nil, fmt.Errorf("%s: %w", string(readMsg.data), ErrHandlerNotFound)
		}
		return nil, fmt.Errorf("peer error: %w", errors.New(readMsg.msg))
	}

	return &Message{
		Data: readMsg.data,
	}, nil
}

func (c *channelQuic) Close() error {
	if c.release != nil {
		c.release()
	}

	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close QUIC connection: %w", err)
	}

	return nil
}

func (c *channelQuic) setTracer(tracer *trace.Tracer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tracer = tracer
}

func (c *channelQuic) setServiceConn(conn ServiceConn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tr = conn.(*streams.QuicConnection)
}

func (c *channelQuic) setUpnpPortsRelease(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.release = f
}

func (c *channelQuic) launchReadSendLoops() error {
	go func() {
		for {
			select {
			case <-c.conn.Context().Done():
				return
			default:
			}

			s, err := c.conn.AcceptStream(context.TODO())
			if err != nil {
				log.Warn().Err(err).Msg("Failed to accept QUIC stream")

				continue
			}

			readMsg := transportMsg{}
			if err := readMsg.readFrom(newCompatibleWireReader(s, c.compatibility)); err != nil {
				log.Warn().Err(err).Msg("Failed to read from QUIC stream")

				continue
			}

			log.Debug().Str("topic", readMsg.topic).Msg("Received message on QUIC stream")

			if readMsg.topic != "" {
				go c.handleRequest(s, &readMsg)
			}
		}
	}()

	return nil
}

func (c *channelQuic) setPeerID(identity identity.Identity) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.id = identity
}

func (c *channelQuic) handleRequest(s quic.Stream, msg *transportMsg) {
	defer func() {
		s.Close()
		s.CancelRead(0)
	}()

	resp, err := c.handleStream(*msg, msg.data)
	if err != nil {
		log.Error().Err(err).Msg("Failed to handle stream")
		return
	}

	if err := resp.writeTo(newCompatibleWireWriter(s, c.compatibility)); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
		return
	}
}

func (c *channelQuic) handleStream(msg transportMsg, buf []byte) (resp *transportMsg, err error) {
	var resMsg transportMsg
	resMsg.id = msg.id

	handler, ok := c.handlers[msg.topic]
	if !ok {
		return nil, fmt.Errorf("handler %q not found", msg.topic)
	}

	ctx := defaultContext{
		req: &Message{
			Data: buf[:],
		},
		peerID: c.id,
	}

	if err = handler(&ctx); err != nil {
		log.Error().Err(err).Msgf("Handler '%q' internal error", msg.topic)

		resMsg.statusCode = statusCodeInternalErr
		resMsg.msg = err.Error()
	} else if ctx.publicError != nil {
		log.Error().Err(ctx.publicError).Msgf("Handler '%q' public error", msg.topic)

		resMsg.statusCode = statusCodePublicErr
		resMsg.data = []byte(ctx.publicError.Error())
	} else {
		resMsg.statusCode = statusCodeOK
		if ctx.res != nil {
			resMsg.data = ctx.res.Data
		}
	}

	return &resMsg, nil
}
