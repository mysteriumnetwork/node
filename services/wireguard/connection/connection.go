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
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/nat/traversal"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Connection which does wireguard tunneling.
type Connection struct {
	done             chan struct{}
	statsCheckerStop chan struct{}
	pingerStop       chan struct{}

	privateKey          string
	ipResolver          ip.Resolver
	stateChannel        connection.StateChannel
	statisticsChannel   connection.StatisticsChannel
	connectionEndpoint  wg.ConnectionEndpoint
	removeAllowedIPRule func()
	configDir           string
	natPinger           traversal.NATProviderPinger
}

// Start establish wireguard connection to the service provider.
func (c *Connection) Start(options connection.ConnectOptions) (err error) {
	var config wg.ServiceConfig
	if err := json.Unmarshal(options.SessionConfig, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal connection config")
	}

	removeAllowedIPRule, err := firewall.AllowIPAccess(config.Provider.Endpoint.IP.String())
	if err != nil {
		return errors.Wrap(err, "failed to add firewall exception for wireguard remote IP")
	}
	c.removeAllowedIPRule = removeAllowedIPRule

	// TODO: Fix conn cleanups https://github.com/mysteriumnetwork/node/issues/1499
	defer func() {
		if err != nil {
			c.stateChannel <- connection.NotConnected
			close(c.done)
			removeAllowedIPRule()
		}
	}()

	natPunchingEnabled := config.LocalPort > 0
	if natPunchingEnabled {
		err = c.natPinger.PingProvider(
			config.Provider.Endpoint.IP.String(),
			config.RemotePort,
			config.LocalPort,
			0,
			c.pingerStop,
		)
		if err != nil {
			return errors.Wrap(err, "could not ping provider")
		}
	}

	c.stateChannel <- connection.Connecting

	log.Info().Msg("Starting new connection")
	conn, err := c.startConn(wg.ConsumerModeConfig{
		PrivateKey: c.privateKey,
		IPAddress:  config.Consumer.IPAddress,
		ListenPort: config.LocalPort,
	})
	if err != nil {
		return errors.Wrap(err, "could not start new connection")
	}
	c.connectionEndpoint = conn

	log.Info().Msg("Adding connection peer")

	if err := c.addProviderPeer(conn, config.Provider.Endpoint, config.Provider.PublicKey); err != nil {
		return errors.Wrap(err, "failed to add peer to the connection endpoint")
	}

	log.Info().Msg("Configuring routes")
	if err := conn.ConfigureRoutes(config.Provider.Endpoint.IP); err != nil {
		return errors.Wrap(err, "failed to configure routes for connection endpoint")
	}

	log.Info().Msg("Waiting for initial handshake")
	if err := wg.WaitHandshake(conn.PeerStats, c.done); err != nil {
		return errors.Wrap(err, "failed while waiting for a peer handshake")
	}

	dnsIPs, err := options.DNS.ResolveIPs(config.Consumer.DNSIPs)
	if err != nil {
		return errors.Wrap(err, "could not resolve DNS IPs")
	}
	config.Consumer.DNSIPs = dnsIPs[0]
	if err := setDNS(c.configDir, conn.InterfaceName(), config.Consumer.DNSIPs); err != nil {
		return errors.Wrap(err, "failed to configure DNS")
	}

	go c.updateStatsPeriodically(time.Second)

	c.stateChannel <- connection.Connected
	return nil
}

func (c *Connection) startConn(conf wg.ConsumerModeConfig) (wg.ConnectionEndpoint, error) {
	resourceAllocator := connectionResourceAllocator()
	conn, err := endpoint.NewConnectionEndpoint(nil, resourceAllocator, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new connection endpoint")
	}

	log.Info().Msg("Starting connection endpoint")
	if err := conn.StartConsumerMode(conf); err != nil {
		return nil, errors.Wrap(err, "failed to start connection endpoint")
	}

	return conn, nil
}

func (c *Connection) addProviderPeer(conn wg.ConnectionEndpoint, endpoint net.UDPAddr, publicKey string) error {
	peerInfo := wg.Peer{
		Endpoint:               &endpoint,
		PublicKey:              publicKey,
		AllowedIPs:             []string{"0.0.0.0/0", "::/0"},
		KeepAlivePeriodSeconds: 18,
	}
	return conn.AddPeer(conn.InterfaceName(), peerInfo)
}

// Wait blocks until wireguard connection not stopped.
func (c *Connection) Wait() error {
	<-c.done
	return nil
}

// GetConfig returns the consumer configuration for session creation
func (c *Connection) GetConfig() (connection.ConsumerConfig, error) {
	publicKey, err := key.PrivateKeyToPublicKey(c.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key from private key")
	}

	var publicIP string
	if !c.isNoopPinger() {
		var err error
		publicIP, err = c.ipResolver.GetPublicIP()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get consumer public IP")
		}
	}
	return wg.ConsumerConfig{
		PublicKey: publicKey,
		IP:        publicIP,
	}, nil
}

func (c *Connection) isNoopPinger() bool {
	_, ok := c.natPinger.(*traversal.NoopPinger)
	return ok
}

// Stop stops wireguard connection and closes connection endpoint.
func (c *Connection) Stop() {
	log.Info().Msg("Stopping WireGuard connection")
	c.stateChannel <- connection.Disconnecting
	c.sendStats()

	if c.connectionEndpoint != nil {
		if err := cleanDNS(c.configDir, c.connectionEndpoint.InterfaceName()); err != nil {
			log.Error().Err(err).Msg("Failed to clear DNS")
		}
		if err := c.connectionEndpoint.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to close wireguard connection")
		}
	}

	if c.removeAllowedIPRule != nil {
		c.removeAllowedIPRule()
	}

	c.stateChannel <- connection.NotConnected
	close(c.done)
	close(c.statsCheckerStop)
	close(c.pingerStop)
	close(c.stateChannel)
	close(c.statisticsChannel)
}

func (c *Connection) updateStatsPeriodically(duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			c.sendStats()
		case <-c.statsCheckerStop:
			return
		}
	}
}

func (c *Connection) sendStats() {
	stats, err := c.connectionEndpoint.PeerStats()
	if err != nil {
		log.Error().Err(err).Msg("Failed to receive peer stats")
		return
	}
	c.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}
}
