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

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/quic/streams"
)

var quicConfig = quic.Config{
	KeepAlivePeriod:       time.Second * 5,
	MaxIdleTimeout:        time.Second * 10,
	MaxIncomingStreams:    10000,
	MaxIncomingUniStreams: 10000,
}

type client struct {
	address string

	mu                sync.RWMutex
	communicationConn quic.Connection
	transportConn     quic.Connection
}

// NewClient creates new QUIC client.
func NewClient(address string) *client {
	return &client{
		address: address,
	}
}

func (c *client) DialCommunication(ctx context.Context) (*streams.QuicConnection, error) {
	protocol := "myst-communication"
	c.mu.Lock()
	defer c.mu.Unlock()

	tlsConf := &tls.Config{NextProtos: []string{protocol}}

	conn, err := c.dial(ctx, tlsConf)
	if err != nil {
		return nil, fmt.Errorf("initial dial failed: %w", err)
	}

	if conn.ConnectionState().TLS.NegotiatedProtocol != "myst-communication" {
		return nil, fmt.Errorf("unexpected protocol: %s", conn.ConnectionState().TLS.NegotiatedProtocol)
	}

	c.communicationConn = conn

	go func() {
		active := true

		for {
			if !active {
				conn, err = c.dial(ctx, tlsConf)
				if err != nil {
					select {
					case <-ctx.Done():
						conn.CloseWithError(100, "stopped")

						return
					default:
						log.Warn().Err(err).Msg("Dial failed, reconnect in 5 seconds")
						time.Sleep(5 * time.Second)

						continue
					}
				} else {
					if conn.ConnectionState().TLS.NegotiatedProtocol != "myst-communication" {
						conn.CloseWithError(300, "unexpected protocol")
						continue
					}

					c.mu.Lock()
					c.communicationConn = conn
					c.mu.Unlock()

					active = true
				}
			}

			select {
			case <-ctx.Done():
				conn.CloseWithError(100, "stopped")

				return
			case <-conn.Context().Done():
				conn.CloseWithError(200, "reconnect")

				active = false
				continue
			}
		}
	}()

	for c.communicationConn == nil {
		log.Info().Msg("Waiting for session")
		time.Sleep(200 * time.Millisecond)
	}

	return &streams.QuicConnection{Connection: c.communicationConn}, nil
}

func (c *client) DialTransport(ctx context.Context) (*streams.QuicConnection, error) {
	protocol := "myst-transport"
	c.mu.Lock()
	defer c.mu.Unlock()

	tlsConf := &tls.Config{NextProtos: []string{protocol}}

	conn, err := c.dial(ctx, tlsConf)
	if err != nil {
		return nil, fmt.Errorf("initial dial failed: %w", err)
	}

	if conn.ConnectionState().TLS.NegotiatedProtocol != "myst-transport" {
		return nil, fmt.Errorf("unexpected protocol: %s", conn.ConnectionState().TLS.NegotiatedProtocol)
	}

	c.transportConn = conn

	go func() {
		active := true

		for {
			if !active {
				conn, err = c.dial(ctx, tlsConf)
				if err != nil {
					select {
					case <-ctx.Done():
						conn.CloseWithError(100, "stopped")

						return
					default:
						log.Warn().Err(err).Msg("Dial failed, reconnect in 5 seconds")
						time.Sleep(5 * time.Second)

						continue
					}
				} else {
					if conn.ConnectionState().TLS.NegotiatedProtocol != "myst-transport" {
						conn.CloseWithError(300, "unexpected protocol")
						continue
					}

					c.mu.Lock()
					c.transportConn = conn
					c.mu.Unlock()

					active = true
				}
			}

			select {
			case <-ctx.Done():
				conn.CloseWithError(100, "stopped")

				return
			case <-conn.Context().Done():
				conn.CloseWithError(200, "reconnect")

				active = false

				continue
			}
		}
	}()

	for c.transportConn == nil {
		log.Info().Msg("Waiting for transport connection")
		time.Sleep(200 * time.Millisecond)
	}

	return &streams.QuicConnection{Connection: c.transportConn}, nil
}

func (c *client) dial(ctx context.Context, tlsConf *tls.Config) (quic.Connection, error) {
	conf := quicConfig

	communicationConn, err := quic.DialAddr(ctx, c.address, tlsConf, &conf)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	return communicationConn, nil
}
