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

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"

	node_config "github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
)

// Options represents connection options.
type Options struct{}

type connectionConfig struct {
	URL string `json:"url"`
}

// NewConnection returns new QUIC connection.
func NewConnection(opts Options) (connection.Connection, error) {
	return &Connection{
		done:    make(chan struct{}),
		stateCh: make(chan connectionstate.State, 100),
		opts:    opts,
	}, nil
}

// Connection which does QUIC tunneling.
type Connection struct {
	stopOnce sync.Once
	done     chan struct{}
	stateCh  chan connectionstate.State

	opts Options

	server *Server
}

var _ connection.Connection = &Connection{}

// State returns connection state channel.
func (c *Connection) State() <-chan connectionstate.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Connection) Statistics() (connectionstate.Statistics, error) {
	in, out := c.server.Stats()
	return connectionstate.Statistics{
		At:            time.Now(),
		BytesSent:     out,
		BytesReceived: in,
	}, nil
}

// Start establish QUIC connection to the service provider.
func (c *Connection) Start(ctx context.Context, options connection.ConnectOptions) error {
	return c.start(ctx, options)
}

// Reconnect restarts a connection with a new options.
func (c *Connection) Reconnect(ctx context.Context, options connection.ConnectOptions) error {
	return c.start(ctx, options)
}

func (c *Connection) start(ctx context.Context, options connection.ConnectOptions) (err error) {
	var config string
	if err = json.Unmarshal(options.SessionConfig, &config); err != nil {
		return fmt.Errorf("failed to unmarshal connection config: %w", err)
	}

	defer func() {
		if err != nil {
			c.Stop()
		}
	}()

	c.stateCh <- connectionstate.Connecting

	log.Debug().Int("port", options.Params.ProxyPort).Msg("Starting QUIC connection")

	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", options.Params.ProxyPort))

	if options.ProviderNATConn != nil {
		c.server = NewServer(options.ProviderNATConn.(quic.Connection), addr, node_config.GetString(node_config.FlagQUICLogin), node_config.GetString(node_config.FlagQUICPassword))
		if err := c.server.listenAndServeRequests(ctx); err != nil {
			return fmt.Errorf("failed to listen and serve requests: %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to start new connection: %w", err)
	}

	c.stateCh <- connectionstate.Connected

	return nil
}

// GetConfig returns the consumer configuration for session creation
func (c *Connection) GetConfig() (connection.ConsumerConfig, error) {
	return connectionConfig{}, nil
}

// Stop stops QUIC connection and closes connection endpoint.
func (c *Connection) Stop() {
	c.stopOnce.Do(func() {
		log.Info().Msg("Stopping QUIC connection")
		c.stateCh <- connectionstate.Disconnecting

		c.server.Stop()

		c.stateCh <- connectionstate.NotConnected

		close(c.stateCh)
		close(c.done)
	})
}
