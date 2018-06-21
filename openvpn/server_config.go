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
	"github.com/mysterium/node/openvpn/config"
	"github.com/mysterium/node/openvpn/tls"
)

// ServerConfig defines openvpn in server mode configuration structure
type ServerConfig struct {
	*config.GenericConfig
}

// SetServerMode sets a set of options for openvpn to act as server
func (c *ServerConfig) SetServerMode(port int, network, netmask string) {
	c.SetPort(port)
	c.SetParam("server", network, netmask)
	c.SetParam("topology", "subnet")
}

// SetTLSServer add tls-server option to config, also sets dh to none
func (c *ServerConfig) SetTLSServer() {
	c.SetFlag("tls-server")
	c.AddOptions(config.OptionParam("dh", "none"))
}

// SetProtocol adds protocol option (tcp or udp)
func (c *ServerConfig) SetProtocol(protocol string) {
	if protocol == "tcp" {
		c.SetParam("proto", "tcp-server")
	} else if protocol == "udp" {
		c.SetFlag("explicit-exit-notify")
	}
}

// NewServerConfig creates server configuration structure from given basic parameters
func NewServerConfig(
	runtimeDir string,
	configDir string,
	network, netmask string,
	secPrimitives *tls.Primitives,
	port int,
	protocol string,
) *ServerConfig {
	serverConfig := ServerConfig{config.NewConfig(runtimeDir, configDir)}
	serverConfig.SetServerMode(port, network, netmask)
	serverConfig.SetTLSServer()
	serverConfig.SetProtocol(protocol)
	serverConfig.SetTLSCACertificate(secPrimitives.CertificateAuthority.ToPEMFormat())
	serverConfig.SetTLSPrivatePubKeys(
		secPrimitives.ServerCertificate.ToPEMFormat(),
		secPrimitives.ServerCertificate.KeyToPEMFormat(),
	)
	serverConfig.SetTLSCrypt(secPrimitives.PresharedKey.ToPEMFormat())

	serverConfig.SetDevice("tun")
	serverConfig.SetParam("cipher", "AES-256-GCM")
	serverConfig.SetParam("verb", "3")
	serverConfig.SetParam("tls-version-min", "1.2")
	serverConfig.SetFlag("management-client-auth")
	serverConfig.SetParam("verify-client-cert", "none")
	serverConfig.SetParam("tls-cipher", "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384")
	serverConfig.SetParam("reneg-sec", "60")
	serverConfig.SetKeepAlive(10, 60)
	serverConfig.SetPingTimerRemote()
	serverConfig.SetPersistTun()
	serverConfig.SetPersistKey()

	return &serverConfig
}
