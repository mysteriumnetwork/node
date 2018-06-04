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
)

// NewServer constructs new openvpn server instance
func NewServer(openvpnBinary string, generateConfig ServerConfigGenerator, middlewares ...management.Middleware) Process {
	serverConfig := generateConfig()
	return &openvpnProcess{
		config:     serverConfig.GenericConfig,
		management: management.NewManagement(management.LocalhostOnRandomPort, "[server-management] ", middlewares...),
		cmd:        NewCmdWrapper(openvpnBinary, "[server-openvpn] "),
	}
}

// ServerConfigGenerator callback returns generated server config
type ServerConfigGenerator func() *ServerConfig

// NewServerConfigGenerator returns function generating server config and generates required security primitives
func NewServerConfigGenerator(directoryRuntime string, primitives *tls.Primitives, port int, protocol string) ServerConfigGenerator {
	return func() *ServerConfig {
		vpnServerConfig := NewServerConfig(
			directoryRuntime,
			"10.8.0.0", "255.255.255.0",
			primitives,
			port,
			protocol,
		)
		return vpnServerConfig
	}
}
