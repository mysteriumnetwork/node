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

package broker

import (
	"context"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
)

// NewConnection creates a new broker service connection.
func NewConnection() (connection.Connection, error) {
	return &Connection{
		stateCh: make(chan connectionstate.State, 100),
	}, nil
}

// Connection to the broker service.
type Connection struct {
	isRunning        bool
	brokerConnection sync.WaitGroup
	stateCh          chan connectionstate.State
}

var _ connection.Connection = &Connection{}

// State returns connection state channel.
func (c *Connection) State() <-chan connectionstate.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Connection) Statistics() (connectionstate.Statistics, error) {
	return connectionstate.Statistics{At: time.Now()}, nil
}

// Start implements the connection.Connection interface.
func (c *Connection) Start(ctx context.Context, params connection.ConnectOptions) error {
	c.brokerConnection.Add(1)
	c.isRunning = true
	c.stateCh <- connectionstate.Connecting
	c.stateCh <- connectionstate.Connected
	return nil
}

// Wait implements the connection.Connection interface.
func (c *Connection) Wait() error {
	if c.isRunning {
		c.brokerConnection.Wait()
	}
	return nil
}

// Stop implements the connection.Connection interface.
func (c *Connection) Stop() {
	if !c.isRunning {
		return
	}

	c.isRunning = false
	c.stateCh <- connectionstate.Disconnecting
	c.stateCh <- connectionstate.NotConnected
	c.brokerConnection.Done()
	close(c.stateCh)
}

// GetConfig returns the consumer configuration for session creation
func (c *Connection) GetConfig() (connection.ConsumerConfig, error) {
	return nil, nil
}
