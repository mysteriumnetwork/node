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

package connection

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/connection"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	endpoint "github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/pkg/errors"
)

const logPrefix = "[connection-wireguard] "

// Connection which does wireguard tunneling.
type Connection struct {
	connection  sync.WaitGroup
	stopChannel chan struct{}

	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel

	config             wg.ServiceConfig
	connectionEndpoint wg.ConnectionEndpoint
}

// Start establish wireguard connection to the service provider.
func (c *Connection) Start(options connection.ConnectOptions) (err error) {
	var config wg.ServiceConfig
	if err := json.Unmarshal(options.SessionConfig, &config); err != nil {
		return err
	}
	c.config.Provider = config.Provider
	c.config.Consumer.IPAddress = config.Consumer.IPAddress

	resourceAllocator := resources.NewAllocator()
	c.connectionEndpoint, err = endpoint.NewConnectionEndpoint("", &resourceAllocator)
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
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		return err
	}

	if err := c.connectionEndpoint.ConfigureRoutes(c.config.Provider.Endpoint.IP); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		return err
	}

	if err := c.waitHandshake(); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		return errors.Wrap(err, "failed while waiting for a peer handshake")
	}

	go c.runPeriodically(time.Second)

	c.stateChannel <- connection.Connected
	return nil
}

// Wait blocks until wireguard connection not stopped.
func (c *Connection) Wait() error {
	c.connection.Wait()
	return nil
}

// GetConfig returns the consumer configuration for session creation
func (c *Connection) GetConfig() (connection.ConsumerConfig, error) {
	publicKey, err := key.PrivateKeyToPublicKey(c.config.Consumer.PrivateKey)
	if err != nil {
		return nil, err
	}
	return wg.ConsumerConfig{
		PublicKey: publicKey,
	}, nil
}

// Stop stops wireguard connection and closes connection endpoint.
func (c *Connection) Stop() {
	c.stateChannel <- connection.Disconnecting

	if err := c.connectionEndpoint.Stop(); err != nil {
		log.Error(logPrefix, "Failed to close wireguard connection: ", err)
	}

	c.stateChannel <- connection.NotConnected
	c.connection.Done()
	close(c.stopChannel)
	close(c.stateChannel)
	close(c.statisticsChannel)
}

func (c *Connection) runPeriodically(duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			stats, _, err := c.connectionEndpoint.PeerStats()
			if err != nil {
				log.Error(logPrefix, "failed to receive peer stats: ", err)
				break
			}
			c.statisticsChannel <- stats

		case <-c.stopChannel:
			return
		}
	}
}

func (c *Connection) waitHandshake() error {
	// We need to send any packet to initialize handshake process
	_, _ = net.DialTimeout("tcp", "8.8.8.8:53", 100*time.Millisecond)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			_, lastHandshake, err := c.connectionEndpoint.PeerStats()
			if err != nil {
				return err
			}
			if lastHandshake != 0 {
				return nil
			}

		case <-c.stopChannel:
			return errors.New("stop received")
		}
	}
}
