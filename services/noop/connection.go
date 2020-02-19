/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
)

// NewConnection creates a new noop connnection
func NewConnection() (connection.Connection, error) {
	return &Connection{
		stateCh: make(chan connection.State, 100),
	}, nil
}

// Connection which does no real tunneling
type Connection struct {
	isRunning      bool
	noopConnection sync.WaitGroup
	stateCh        chan connection.State
}

var _ connection.Connection = &Connection{}

// State returns connection state channel.
func (c *Connection) State() <-chan connection.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Connection) Statistics() (connection.Statistics, error) {
	return connection.Statistics{At: time.Now()}, nil
}

// Start implements the connection.Connection interface
func (c *Connection) Start(params connection.ConnectOptions) error {
	c.noopConnection.Add(1)
	c.isRunning = true

	c.stateCh <- connection.Connecting

	time.Sleep(5 * time.Second)
	c.stateCh <- connection.Connected
	return nil
}

// Wait implements the connection.Connection interface
func (c *Connection) Wait() error {
	if c.isRunning {
		c.noopConnection.Wait()
	}
	return nil
}

// Stop implements the connection.Connection interface
func (c *Connection) Stop() {
	if !c.isRunning {
		return
	}

	c.isRunning = false
	c.stateCh <- connection.Disconnecting
	time.Sleep(2 * time.Second)
	c.stateCh <- connection.NotConnected
	c.noopConnection.Done()
	close(c.stateCh)
}

// GetConfig returns the consumer configuration for session creation
func (c *Connection) GetConfig() (connection.ConsumerConfig, error) {
	return nil, nil
}
