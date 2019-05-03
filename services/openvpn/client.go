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

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/pkg/errors"
)

// ErrProcessNotStarted represents the error we return when the process is not started yet
var ErrProcessNotStarted = errors.New("process not started yet")

// processFactory creates a new openvpn process
type processFactory func(options connection.ConnectOptions) (openvpn.Process, *ClientConfig, error)

// NATPinger tries to punch a hole in NAT
type NATPinger interface {
	BindConsumerPort(port int)
	Stop()
	PingProvider(ip string, port int, stop <-chan struct{}) error
}

// Client takes in the openvpn process and works with it
type Client struct {
	process        openvpn.Process
	processFactory processFactory
	ipResolver     ip.Resolver
	natPinger      NATPinger
	publicIP       string
	pingerStop     chan struct{}

	stopOnce sync.Once
}

// Start starts the connection
func (c *Client) Start(options connection.ConnectOptions) error {
	log.Info("starting connection")
	proc, clientConfig, err := c.processFactory(options)
	if err != nil {
		log.Info("client config factory error: ", err)
		return err
	}
	c.process = proc
	log.Infof("client config: %v", clientConfig)

	c.natPinger.BindConsumerPort(clientConfig.LocalPort)
	if clientConfig.VpnConfig.LocalPort > 0 {
		err = c.natPinger.PingProvider(clientConfig.VpnConfig.RemoteIP, clientConfig.VpnConfig.RemotePort, c.pingerStop)
		if err != nil {
			return err
		}
	}
	return c.process.Start()
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
		log.Infof("noop pinger detected, returning nil client config.")
		return nil, nil
	}

	return &ConsumerConfig{
		// TODO: since GetConfig is executed before Start we cannot access VPNConfig structure yet
		// TODO skip sending port here, since provider generates port for consumer in VPNConfig
		//Port: c.vpnClientConfig.LocalPort,
		Port: 50221,
		IP:   &ip,
	}, nil
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	LocalPort       int    `json:"lport"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}
