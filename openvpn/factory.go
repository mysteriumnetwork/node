package openvpn

import (
	"github.com/mysterium/node/openvpn/primitives"
	"github.com/mysterium/node/session"
	"io/ioutil"
)

func NewServerConfig(
	network, netmask string,
	secPrimitives *primitives.SecurityPrimitives,
	port int,
	protocol string,
) *ServerConfig {
	config := ServerConfig{NewConfig()}
	config.SetServerMode(port, network, netmask)
	config.SetTLSServer()
	config.SetProtocol(protocol)
	config.SetTLSCACertificate(secPrimitives.CACertPath)
	config.SetTLSPrivatePubKeys(secPrimitives.ServerCertPath, secPrimitives.ServerKeyPath)
	config.SetTLSCrypt(secPrimitives.TLSCryptKeyPath)

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

func newClientConfig(vpnConfig session.VPNConfig) *ClientConfig {
	config := ClientConfig{NewConfig()}
	config.SetClientMode(vpnConfig.RemoteIP, vpnConfig.RemotePort)
	config.SetProtocol(vpnConfig.RemoteProtocol)
	config.SetTLSCACertificate(vpnConfig.CACertificate)
	config.SetTLSCrypt(vpnConfig.TLSPresharedKey)
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
	config.setParam("redirect-gateway", "def1 bypass-dhcp")
	config.setParam("dhcp-option", "DNS 208.67.222.222")
	config.setParam("dhcp-option", "DNS 208.67.220.220")

	return &config
}

func NewClientConfigFromSession(vpnConfig session.VPNConfig, configFile string) (*ClientConfig, error) {

	err := NewDefaultValidator().IsValid(vpnConfig)
	if err != nil {
		return nil, err
	}

	clientConfig := newClientConfig(vpnConfig)
	configAsString, err := ConfigToString(*clientConfig.Config)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(configFile, []byte(configAsString), 0600)
	if err != nil {
		return nil, err
	}

	config := ClientConfig{NewConfig()}
	config.AddOptions(OptionParam("config", configFile))

	config.setParam("up", "update-resolv-conf")
	config.setParam("down", "update-resolv-conf")

	return &config, nil
}
