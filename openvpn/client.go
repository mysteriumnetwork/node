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
	"github.com/mysterium/node/openvpn/management"
	"github.com/mysterium/node/openvpn/tls"
	"github.com/mysterium/node/session"
)

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, middlewares ...management.Middleware) Process {

	return &openvpnProcess{
		config:     config.GenericConfig,
		management: management.NewManagement(management.LocalhostOnRandomPort, "[client-management] ", middlewares...),
		cmd:        NewCmdWrapper(openvpnBinary, "[client-openvpn] "),
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

// ProvideServiceConfig callback providing service configuration for a session
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
