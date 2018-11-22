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

package wireguard

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
)

// Connection which does no real tunneling
type Connection struct {
	isRunning      bool
	noopConnection sync.WaitGroup
	stateChannel   connection.StateChannel
	statsChannel   connection.StatsChannel
}

// Start implements the connection.Connection interface
func (c *Connection) Start() error {
	c.noopConnection.Add(1)
	c.isRunning = true

	c.stateChannel <- connection.Connecting

	// TODO establish real wireguard connection to the service provider
	time.Sleep(5 * time.Second)

	c.stateChannel <- connection.Connected
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
	c.stateChannel <- connection.Disconnecting

	// TODO destroy wireguard connection
	time.Sleep(2 * time.Second)

	c.stateChannel <- connection.NotConnected
	c.noopConnection.Done()
	close(c.stateChannel)
}
