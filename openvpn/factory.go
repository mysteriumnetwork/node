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
	serverConfig := ServerConfig{config.NewConfig(configDir)}
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

func newClientConfig(configDir string) *ClientConfig {
	clientConfig := ClientConfig{config.NewConfig(configDir)}

	clientConfig.RestrictReconnects()

	clientConfig.SetDevice("tun")
	clientConfig.SetParam("cipher", "AES-256-GCM")
	clientConfig.SetParam("verb", "3")
	clientConfig.SetParam("tls-cipher", "TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384")
	clientConfig.SetKeepAlive(10, 60)
	clientConfig.SetPingTimerRemote()
	clientConfig.SetPersistTun()
	clientConfig.SetPersistKey()

	clientConfig.SetParam("reneg-sec", "60")
	clientConfig.SetParam("resolv-retry", "infinite")
	clientConfig.SetParam("redirect-gateway", "def1", "bypass-dhcp")
	clientConfig.SetParam("dhcp-option", "DNS", "208.67.222.222")
	clientConfig.SetParam("dhcp-option", "DNS", "208.67.220.220")

	return &clientConfig
}

func NewClientConfigFromSession(vpnConfig *VPNConfig, configDir string, configFile string) (*ClientConfig, error) {

	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientFileConfig := newClientConfig(configDir)
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

	clientConfig := ClientConfig{config.NewConfig(configDir)}
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
