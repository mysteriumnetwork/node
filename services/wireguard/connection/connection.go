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

	"github.com/mysteriumnetwork/node/firewall"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	endpoint "github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
)

const logPrefix = "[connection-wireguard] "

// Connection which does wireguard tunneling.
type Connection struct {
	connection  sync.WaitGroup
	stopChannel chan struct{}

	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel

	config              wg.ServiceConfig
	connectionEndpoint  wg.ConnectionEndpoint
	removeAllowedIPRule func()
}

// Start establish wireguard connection to the service provider.
func (c *Connection) Start(options connection.ConnectOptions) (err error) {
	var config wg.ServiceConfig
	if err := json.Unmarshal(options.SessionConfig, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal connection config")
	}
	c.config.Provider = config.Provider
	c.config.Consumer.IPAddress = config.Consumer.IPAddress
	// TODO its funny that remote IP of wireguard provider is called config.Consumer.IPAddress
	removeAllowedIPRule, err := firewall.AllowIPAccess(config.Consumer.IPAddress.IP.String())
	if err != nil {
		return errors.Wrap(err, "failed to add firewall exception for wireguard remote IP")
	}
	c.removeAllowedIPRule = removeAllowedIPRule

	// We do not need port mapping for consumer, since it initiates the session
	fakePortMapper := func(port int) (releasePortMapping func()) {
		return func() {}
	}

	resourceAllocator := connectionResourceAllocator()
	c.connectionEndpoint, err = endpoint.NewConnectionEndpoint(nil, resourceAllocator, fakePortMapper, 0)
	if err != nil {
		removeAllowedIPRule()
		return errors.Wrap(err, "failed to create new connection endpoint")
	}

	c.connection.Add(1)
	c.stateChannel <- connection.Connecting

	if err := c.connectionEndpoint.Start(&c.config); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		removeAllowedIPRule()
		return errors.Wrap(err, "failed to start connection endpoint")
	}

	// Provider requests to delay consumer connection since it might be in a process of setting up NAT traversal for given consumer
	if config.Consumer.ConnectDelay > 0 {
		log.Infof("%s delaying connect for %v milliseconds", logPrefix, config.Consumer.ConnectDelay)
		time.Sleep(time.Duration(config.Consumer.ConnectDelay) * time.Millisecond)
	}

	if err := c.connectionEndpoint.AddPeer(c.config.Provider.PublicKey, &c.config.Provider.Endpoint); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		removeAllowedIPRule()
		return errors.Wrap(err, "failed to add peer to the connection endpoint")
	}

	if err := c.connectionEndpoint.ConfigureRoutes(c.config.Provider.Endpoint.IP); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		removeAllowedIPRule()
		return errors.Wrap(err, "failed to configure routes for connection endpoint")
	}

	if err := c.waitHandshake(); err != nil {
		c.stateChannel <- connection.NotConnected
		c.connection.Done()
		removeAllowedIPRule()
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
	c.sendStats()

	if err := c.connectionEndpoint.Stop(); err != nil {
		log.Error(logPrefix, "Failed to close wireguard connection: ", err)
	}
	c.removeAllowedIPRule()
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
			c.sendStats()

		case <-c.stopChannel:
			return
		}
	}
}

func (c *Connection) sendStats() {
	stats, err := c.connectionEndpoint.PeerStats()
	if err != nil {
		log.Error(logPrefix, "failed to receive peer stats: ", err)
		return
	}
	c.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}
}

func (c *Connection) waitHandshake() error {
	// We need to send any packet to initialize handshake process
	_, _ = net.DialTimeout("tcp", "8.8.8.8:53", 100*time.Millisecond)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			stats, err := c.connectionEndpoint.PeerStats()
			if err != nil {
				return err
			}
			if !stats.LastHandshake.IsZero() {
				return nil
			}

		case <-c.stopChannel:
			return errors.New("stop received")
		}
	}
}
