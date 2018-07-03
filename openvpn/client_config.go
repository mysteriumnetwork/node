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
	"io/ioutil"
	"path/filepath"
)

// ClientConfig represents specific "openvpn as client" configuration
type ClientConfig struct {
	*config.GenericConfig
}

// SetClientMode adds config arguments for openvpn behave as client
func (c *ClientConfig) SetClientMode(serverIP string, serverPort int) {
	c.SetFlag("client")
	c.SetParam("script-security", "2")
	c.SetFlag("auth-nocache")
	c.SetParam("remote", serverIP)
	c.SetPort(serverPort)
	c.SetFlag("nobind")
	c.SetParam("remote-cert-tls", "server")
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

func newClientConfig(runtimeDir string, scriptSearchPath string) *ClientConfig {
	clientConfig := ClientConfig{config.NewConfig(runtimeDir, scriptSearchPath)}

	clientConfig.RestrictReconnects()

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
func NewClientConfigFromSession(vpnConfig *VPNConfig, configDir string, runtimeDir string) (*ClientConfig, error) {

	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientFileConfig := newClientConfig(runtimeDir, configDir)
	clientFileConfig.SetClientMode(vpnConfig.RemoteIP, vpnConfig.RemotePort)
	clientFileConfig.SetProtocol(vpnConfig.RemoteProtocol)
	clientFileConfig.SetTLSCACertificate(vpnConfig.CACertificate)
	clientFileConfig.SetTLSCrypt(vpnConfig.TLSPresharedKey)

	configAsString, err := clientFileConfig.ToConfigFileContent()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(runtimeDir, "client.ovpn")
	err = ioutil.WriteFile(configFile, []byte(configAsString), 0600)
	if err != nil {
		return nil, err
	}

	clientConfig := ClientConfig{config.NewConfig(runtimeDir, configDir)}
	clientConfig.AddOptions(config.OptionFile("config", configAsString, configFile))

	clientConfig.SetScriptParam("up", config.QuotedPath("update-resolv-conf"))
	clientConfig.SetScriptParam("down", config.QuotedPath("update-resolv-conf"))

	return &clientConfig, nil
}
