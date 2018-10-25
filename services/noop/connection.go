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
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
)

// Connection which does no real tunneling
type Connection struct {
	stateChannel connection.StateChannel
}

// Start implements the connection.Connection interface
func (c *Connection) Start() error {
	c.stateChannel <- connection.Connecting
	time.Sleep(5 * time.Second)
	c.stateChannel <- connection.Connected

	return nil
}

// Wait implements the connection.Connection interface
func (c *Connection) Wait() error {
	return nil
}

// Stop implements the connection.Connection interface
func (c *Connection) Stop() {
	c.stateChannel <- connection.Disconnecting
	time.Sleep(2 * time.Second)
	c.stateChannel <- connection.NotConnected
}
