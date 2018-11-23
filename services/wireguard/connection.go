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
	"github.com/mysteriumnetwork/node/services/wireguard/network"
)

// Connection which does no real tunneling
type Connection struct {
	isRunning    bool
	connection   sync.WaitGroup
	stateChannel connection.StateChannel

	config Config
	wg     Consumer
}

// Consumer represents Wireguard network instance that consume provided service.
type Consumer interface {
	Consumer(provider network.Provider, consumer network.Consumer) error
	Close() error
}

// Start implements the connection.Connection interface
func (c *Connection) Start() error {
	wg, err := network.NewNetwork(interfaceName, "")
	if err != nil {
		return err
	}
	c.wg = wg

	c.connection.Add(1)
	c.isRunning = true

	c.stateChannel <- connection.Connecting

	if err := c.wg.Consumer(c.config.Provider, c.config.Consumer); err != nil {
		c.isRunning = false
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		return err
	}

	c.stateChannel <- connection.Connected
	return nil
}

// Wait implements the connection.Connection interface
func (c *Connection) Wait() error {
	if c.isRunning {
		c.connection.Wait()
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

	if err := c.wg.Close(); err != nil {
		log.Error(logPrefix, "Failed to close wireguard connection", err)
	}

	c.stateChannel <- connection.NotConnected
	c.connection.Done()
	close(c.stateChannel)
}
