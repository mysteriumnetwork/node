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
	"strconv"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/config"
)

// ClientConfig represents specific "openvpn as client" configuration
type ClientConfig struct {
	*config.GenericConfig
	LocalPort int
	VpnConfig *VPNConfig
}

// SetClientMode adds config arguments for openvpn behave as client
func (c *ClientConfig) SetClientMode(serverIP string, serverPort, localPort int) {
	c.SetFlag("client")
	c.SetParam("script-security", "2")
	c.SetFlag("auth-nocache")
	c.SetParam("remote", serverIP)
	c.SetPort(serverPort)
	c.SetParam("lport", strconv.Itoa(localPort))
	c.SetFlag("float")
	// more on this: https://www.v13.gr/blog/?p=386
	c.SetParam("remote-cert-ku", "84")
	c.LocalPort = localPort
	c.SetFlag("auth-user-pass")
	c.SetFlag("management-query-passwords")
}

// SetProtocol specifies openvpn connection protocol type (tcp or udp)
func (c *ClientConfig) SetProtocol(protocol string) {
	if protocol == "tcp" {
		c.SetParam("proto", "tcp-client")
	} else if protocol == "udp" {
		c.SetFlag("explicit-exit-notify")
	}
}

func defaultClientConfig(runtimeDir string, scriptSearchPath string) *ClientConfig {
	clientConfig := ClientConfig{GenericConfig: config.NewConfig(runtimeDir, scriptSearchPath), VpnConfig: nil}

	clientConfig.SetDevice("tun")
	clientConfig.SetParam("cipher", "AES-256-GCM")
	clientConfig.SetParam("verb", "3")
	clientConfig.SetParam("tls-cipher", "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384")
	clientConfig.SetKeepAlive(10, 60)
	clientConfig.SetPingTimerRemote()
	clientConfig.SetPersistKey()

	clientConfig.SetParam("reneg-sec", "60")
	clientConfig.SetParam("resolv-retry", "infinite")
	clientConfig.SetParam("redirect-gateway", "def1", "bypass-dhcp")
	clientConfig.SetParam("dhcp-option", "DNS", "208.67.222.222")
	clientConfig.SetParam("dhcp-option", "DNS", "208.67.220.220")

	return &clientConfig
}

// NewClientConfigFromSession creates client configuration structure for given VPNConfig, configuration dir to store serialized file args, and
// configuration filename to store other args
// TODO this will become the part of openvpn service consumer separate package
func NewClientConfigFromSession(vpnConfig *VPNConfig, configDir string, runtimeDir string, isMobile bool) (*ClientConfig, error) {
	// TODO Rename `vpnConfig` to `sessionConfig`
	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientFileConfig := newClientConfig(runtimeDir, configDir)

	// override vpnClientConfig params with proxy local IP and pinger port
	// do this only if connecting to natted provider
	if vpnConfig.LocalPort > 0 {
		vpnConfig.OriginalRemoteIP = vpnConfig.RemoteIP
		vpnConfig.OriginalRemotePort = vpnConfig.RemotePort
		if isMobile {
			vpnConfig.RemoteIP = "127.0.0.1"
			// TODO: randomize this too?
			vpnConfig.RemotePort = vpnConfig.LocalPort + 1
		}
	}

	clientFileConfig.VpnConfig = vpnConfig
	clientFileConfig.SetReconnectRetry(2)
	clientFileConfig.SetClientMode(vpnConfig.RemoteIP, vpnConfig.RemotePort, vpnConfig.LocalPort)
	clientFileConfig.SetProtocol(vpnConfig.RemoteProtocol)
	clientFileConfig.SetTLSCACertificate(vpnConfig.CACertificate)
	clientFileConfig.SetTLSCrypt(vpnConfig.TLSPresharedKey)

	return clientFileConfig, nil
}
