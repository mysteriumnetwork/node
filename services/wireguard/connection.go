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

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/connection"
)

// Connection which does no real tunneling
type Connection struct {
	connection   sync.WaitGroup
	stateChannel connection.StateChannel

	config             serviceConfig
	connectionEndpoint ConnectionEndpoint
}

// Start implements the connection.Connection interface
func (c *Connection) Start() (err error) {
	c.connectionEndpoint, err = NewConnectionEndpoint(nil)
	if err != nil {
		return err
	}

	c.connection.Add(1)
	c.stateChannel <- connection.Connecting

	if err := c.connectionEndpoint.Start(&c.config); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		return err
	}

	if err := c.connectionEndpoint.AddPeer(c.config.Provider.PublicKey, &c.config.Provider.Endpoint); err != nil {
		return err
	}
	c.stateChannel <- connection.Connected
	return nil
}

// Wait implements the connection.Connection interface
func (c *Connection) Wait() error {
	c.connection.Wait()
	return nil
}

// Stop implements the connection.Connection interface
func (c *Connection) Stop() {
	c.stateChannel <- connection.Disconnecting

	if err := c.connectionEndpoint.Stop(); err != nil {
		log.Error(logPrefix, "Failed to close wireguard connection", err)
	}

	c.stateChannel <- connection.NotConnected
	c.connection.Done()
	close(c.stateChannel)
}
