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

import "sync"
import (
	"errors"
	"github.com/mysterium/node/openvpn/management"
	"github.com/mysterium/node/openvpn/tls"
	"github.com/mysterium/node/session"
	"time"
)

// Client defines client process interfaces with basic commands
type Client interface {
	Start() error
	Wait() error
	Stop() error
}

type openVpnClient struct {
	config     *ClientConfig
	management *management.Management
	process    *Process
}

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, middlewares ...management.Middleware) *openVpnClient {

	return &openVpnClient{
		config:     config,
		management: management.NewManagement(management.LocalhostOnRandomPort, "[client-management] ", middlewares...),
		process:    NewProcess(openvpnBinary, "[client-openvpn] "),
	}
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}

// ClientConfigGenerator callback returns generated server config
type ClientConfigGenerator func() *VPNConfig

func (generator ClientConfigGenerator) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return generator(), nil
}

// NewClientConfigGenerator returns function generating config params for remote client
func NewClientConfigGenerator(primitives *tls.Primitives, vpnServerIP string, port int, protocol string) ClientConfigGenerator {
	return func() *VPNConfig {
		return &VPNConfig{
			vpnServerIP,
			port,
			protocol,
			primitives.PresharedKey.ToPEMFormat(),
			primitives.CertificateAuthority.ToPEMFormat(),
		}
	}
}

func (client *openVpnClient) Start() error {
	err := client.management.WaitForConnection()
	if err != nil {
		return err
	}

	addr := client.management.BoundAddress
	client.config.SetManagementAddress(addr.IP, addr.Port)

	// Fetch the current arguments
	arguments, err := (*client.config).ToArguments()
	if err != nil {
		return err
	}

	//nil returned from process.Start doesn't guarantee that openvpn itself initialized correctly and accepted all arguments
	//it simply means that OS started process with specified args
	err = client.process.Start(arguments)
	if err != nil {
		client.management.Stop()
		return err
	}

	select {
	case connAccepted := <-client.management.Connected:
		if connAccepted {
			return nil
		}
		return errors.New("management failed to accept connection")
	case <-time.After(2 * time.Second):
		return errors.New("management connection wait timeout")
	}
}

func (client *openVpnClient) Wait() error {
	return client.process.Wait()
}

func (client *openVpnClient) Stop() error {
	waiter := sync.WaitGroup{}
	//TODO which to signal for close first ?
	//if we stop process before management, managemnt won't have a chance to send any commands from middlewares on stop
	//if we stop management first - it will miss important EXITING state from process
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		client.process.Stop()
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		client.management.Stop()
	}()

	waiter.Wait()
	return nil
}
