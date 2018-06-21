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
	"fmt"
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

func newClientConfig(configDir string, scriptSearchPath string) *ClientConfig {
	clientConfig := ClientConfig{config.NewConfig(configDir, scriptSearchPath)}

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
func NewClientConfigFromSession(vpnConfig *VPNConfig, configDir string, configFile string) (*ClientConfig, error) {

	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientFileConfig := newClientConfig(configDir, "")
	clientFileConfig.SetClientMode(vpnConfig.RemoteIP, vpnConfig.RemotePort)
	clientFileConfig.SetProtocol(vpnConfig.RemoteProtocol)
	clientFileConfig.SetTLSCACertificate(vpnConfig.CACertificate)
	clientFileConfig.SetTLSCrypt(vpnConfig.TLSPresharedKey)

	configAsString, err := clientFileConfig.ToConfigFileContent()
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(configFile, []byte(configAsString), 0600)
	if err != nil {
		return nil, err
	}

	clientConfig := ClientConfig{config.NewConfig(configDir, "")}
	clientConfig.AddOptions(config.OptionFile("config", configAsString, configFile))

	//because of special case how openvpn handles executable/scripts paths, we need to surround values with double quotes
	updateResolvConfScriptPath := wrapWithDoubleQuotes(filepath.Join(configDir, "update-resolv-conf"))

	clientConfig.SetParam("up", updateResolvConfScriptPath)
	clientConfig.SetParam("down", updateResolvConfScriptPath)

	return &clientConfig, nil
}

func wrapWithDoubleQuotes(val string) string {
	return fmt.Sprintf(`"%s"`, val)
}
