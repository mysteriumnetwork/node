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
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Options represents connection options.
type Options struct {
	DNSScriptDir     string
	HandshakeTimeout time.Duration
}

// NewConnection returns new WireGuard connection.
func NewConnection(opts Options, ipResolver ip.Resolver, endpointFactory wg.EndpointFactory, dnsManager DNSManager, handshakeWaiter HandshakeWaiter) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate private key")
	}

	return &Connection{
		done:                make(chan struct{}),
		stateCh:             make(chan connection.State, 100),
		privateKey:          privateKey,
		opts:                opts,
		ipResolver:          ipResolver,
		connEndpointFactory: endpointFactory,
		dnsManager:          dnsManager,
		handshakeWaiter:     handshakeWaiter,
	}, nil
}

// Connection which does wireguard tunneling.
type Connection struct {
	stopOnce sync.Once
	done     chan struct{}
	stateCh  chan connection.State

	ports               []int
	privateKey          string
	ipResolver          ip.Resolver
	connectionEndpoint  wg.ConnectionEndpoint
	removeAllowedIPRule func()
	opts                Options
	connEndpointFactory wg.EndpointFactory
	dnsManager          DNSManager
	handshakeWaiter     HandshakeWaiter
}

var _ connection.Connection = &Connection{}

// State returns connection state channel.
func (c *Connection) State() <-chan connection.State {
	return c.stateCh
}

// Statistics returns connection statistics channel.
func (c *Connection) Statistics() (connection.Statistics, error) {
	stats, err := c.connectionEndpoint.PeerStats()
	if err != nil {
		return connection.Statistics{}, err
	}
	return connection.Statistics{
		At:            time.Now(),
		BytesSent:     stats.BytesSent,
		BytesReceived: stats.BytesReceived,
	}, nil
}

// Start establish wireguard connection to the service provider.
func (c *Connection) Start(ctx context.Context, options connection.ConnectOptions) (err error) {
	var config wg.ServiceConfig
	if err := json.Unmarshal(options.SessionConfig, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal connection config")
	}

	removeAllowedIPRule, err := firewall.AllowIPAccess(config.Provider.Endpoint.IP.String())
	if err != nil {
		return errors.Wrap(err, "failed to add firewall exception for wireguard remote IP")
	}
	c.removeAllowedIPRule = removeAllowedIPRule

	defer func() {
		if err != nil {
			c.Stop()
		}
	}()

	c.stateCh <- connection.Connecting

	if options.ProviderNATConn != nil {
		options.ProviderNATConn.Close()
		config.LocalPort = options.ProviderNATConn.LocalAddr().(*net.UDPAddr).Port
		config.Provider.Endpoint.Port = options.ProviderNATConn.RemoteAddr().(*net.UDPAddr).Port
	}

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

	log.Info().Msgf("Adding connection peer %s", config.Provider.Endpoint.String())

	if err := c.addProviderPeer(conn, config.Provider.Endpoint, config.Provider.PublicKey); err != nil {
		return errors.Wrap(err, "failed to add peer to the connection endpoint")
	}

	log.Info().Msg("Configuring routes")
	if err := conn.ConfigureRoutes(config.Provider.Endpoint.IP); err != nil {
		return errors.Wrap(err, "failed to configure routes for connection endpoint")
	}

	log.Info().Msg("Waiting for initial handshake")
	if err := c.handshakeWaiter.Wait(conn.PeerStats, c.opts.HandshakeTimeout, c.done); err != nil {
		return errors.Wrap(err, "failed while waiting for a peer handshake")
	}

	dnsIPs, err := options.DNS.ResolveIPs(config.Consumer.DNSIPs)
	if err != nil {
		return errors.Wrap(err, "could not resolve DNS IPs")
	}
	config.Consumer.DNSIPs = dnsIPs[0]
	if err := c.dnsManager.Set(c.opts.DNSScriptDir, conn.InterfaceName(), config.Consumer.DNSIPs); err != nil {
		return errors.Wrap(err, "failed to configure DNS")
	}

	c.stateCh <- connection.Connected
	return nil
}

func (c *Connection) startConn(conf wg.ConsumerModeConfig) (wg.ConnectionEndpoint, error) {
	conn, err := c.connEndpointFactory()
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

	return wg.ConsumerConfig{
		PublicKey: publicKey,
		Ports:     c.ports,
	}, nil
}

// Stop stops wireguard connection and closes connection endpoint.
func (c *Connection) Stop() {
	c.stopOnce.Do(func() {
		log.Info().Msg("Stopping WireGuard connection")
		c.stateCh <- connection.Disconnecting

		if c.connectionEndpoint != nil {
			if err := c.dnsManager.Clean(c.opts.DNSScriptDir, c.connectionEndpoint.InterfaceName()); err != nil {
				log.Error().Err(err).Msg("Failed to clear DNS")
			}
			if err := c.connectionEndpoint.Stop(); err != nil {
				log.Error().Err(err).Msg("Failed to close wireguard connection")
			}
		}

		if c.removeAllowedIPRule != nil {
			c.removeAllowedIPRule()
		}

		c.stateCh <- connection.NotConnected

		close(c.stateCh)
		close(c.done)
	})
}
