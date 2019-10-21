/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package openvpn

import (
	"sync"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrProcessNotStarted represents the error we return when the process is not started yet
var ErrProcessNotStarted = errors.New("process not started yet")

// processFactory creates a new openvpn process
type processFactory func(options connection.ConnectOptions) (openvpn.Process, *ClientConfig, error)

// NATPinger tries to punch a hole in NAT
type NATPinger interface {
	Stop()
	PingProvider(ip string, port int, consumerPort int, stop <-chan struct{}) error
}

// Client takes in the openvpn process and works with it
type Client struct {
	process             openvpn.Process
	processFactory      processFactory
	ipResolver          ip.Resolver
	natPinger           NATPinger
	publicIP            string
	pingerStop          chan struct{}
	removeAllowedIPRule func()
	stopOnce            sync.Once
}

// Start starts the connection
func (c *Client) Start(options connection.ConnectOptions) error {
	log.Info().Msg("Starting connection")
	proc, clientConfig, err := c.processFactory(options)
	if err != nil {
		log.Info().Err(err).Msg("Client config factory error")
		return err
	}
	c.process = proc
	log.Info().Msgf("client config: %v", clientConfig)
	removeAllowedIPRule, err := firewall.AllowIPAccess(clientConfig.VpnConfig.RemoteIP)
	if err != nil {
		return err
	}
	c.removeAllowedIPRule = removeAllowedIPRule

	if clientConfig.VpnConfig.LocalPort > 0 {
		err = c.natPinger.PingProvider(
			clientConfig.VpnConfig.OriginalRemoteIP,
			clientConfig.VpnConfig.OriginalRemotePort,
			clientConfig.LocalPort,
			c.pingerStop,
		)
		if err != nil {
			removeAllowedIPRule()
			return err
		}
	}
	err = c.process.Start()
	if err != nil {
		removeAllowedIPRule()
	}
	return err
}

// Wait waits for the connection to exit
func (c *Client) Wait() error {
	if c.process == nil {
		return ErrProcessNotStarted
	}
	return c.process.Wait()
}

// Stop stops the connection
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		if c.process != nil {
			c.process.Stop()
		}
		c.removeAllowedIPRule()
		close(c.pingerStop)
	})
}

// GetConfig returns the consumer-side configuration.
func (c *Client) GetConfig() (connection.ConsumerConfig, error) {
	ip, err := c.ipResolver.GetPublicIP()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get consumer config")
	}
	c.publicIP = ip

	switch c.natPinger.(type) {
	case *traversal.NoopPinger:
		log.Info().Msg("Noop pinger detected, returning nil client config.")
		return nil, nil
	}

	return &ConsumerConfig{
		IP: &ip,
	}, nil
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	OriginalRemoteIP   string
	OriginalRemotePort int

	DNS             string `json:"dns"`
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	LocalPort       int    `json:"lport"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}
