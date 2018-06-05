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
	"github.com/mysterium/node/openvpn/tls"
	"io/ioutil"
	"path/filepath"
)

func NewServerConfig(
	configDir string,
	network, netmask string,
	secPrimitives *tls.Primitives,
	port int,
	protocol string,
) *ServerConfig {
	config := ServerConfig{NewConfig(configDir)}
	config.SetServerMode(port, network, netmask)
	config.SetTLSServer()
	config.SetProtocol(protocol)
	config.SetTLSCACertificate(secPrimitives.CertificateAuthority.ToPEMFormat())
	config.SetTLSPrivatePubKeys(
		secPrimitives.ServerCertificate.ToPEMFormat(),
		secPrimitives.ServerCertificate.KeyToPEMFormat(),
	)
	config.SetTLSCrypt(secPrimitives.PresharedKey.ToPEMFormat())

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-GCM")
	config.setParam("verb", "3")
	config.setParam("tls-version-min", "1.2")
	config.setFlag("management-client-auth")
	config.setParam("verify-client-cert", "none")
	config.setParam("tls-cipher", "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384")
	config.setParam("reneg-sec", "60")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	return &config
}

func newClientConfig(configDir string) *ClientConfig {
	config := ClientConfig{NewConfig(configDir)}

	config.RestrictReconnects()

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-GCM")
	config.setParam("verb", "3")
	config.setParam("tls-cipher", "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	config.setParam("reneg-sec", "60")
	config.setParam("resolv-retry", "infinite")
	config.setParam("redirect-gateway", "def1", "bypass-dhcp")
	config.setParam("dhcp-option", "DNS", "208.67.222.222")
	config.setParam("dhcp-option", "DNS", "208.67.220.220")

	return &config
}

func NewClientConfigFromSession(vpnConfig *VPNConfig, configDir string, configFile string) (*ClientConfig, error) {

	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientConfig := newClientConfig(configDir)
	clientConfig.SetClientMode(vpnConfig.RemoteIP, vpnConfig.RemotePort)
	clientConfig.SetProtocol(vpnConfig.RemoteProtocol)
	clientConfig.SetTLSCACertificate(vpnConfig.CACertificate)
	clientConfig.SetTLSCrypt(vpnConfig.TLSPresharedKey)

	configAsString, err := ConfigToString(*clientConfig.Config)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(configFile, []byte(configAsString), 0600)
	if err != nil {
		return nil, err
	}

	config := ClientConfig{NewConfig(configDir)}
	config.AddOptions(OptionFile("config", configAsString, configFile))

	//because of special case how openvpn handles executable/scripts paths, we need to surround values with double quotes
	updateResolvConfScriptPath := wrapWithDoubleQuotes(filepath.Join(configDir, "update-resolv-conf"))

	config.setParam("up", updateResolvConfScriptPath)
	config.setParam("down", updateResolvConfScriptPath)

	return &config, nil
}

func wrapWithDoubleQuotes(val string) string {
	return fmt.Sprintf(`"%s"`, val)
}
