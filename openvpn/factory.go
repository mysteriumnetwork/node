package openvpn

import (
	"io/ioutil"
)

func NewServerConfig(
	network, netmask string,
	caFile, certFile, certKeyFile,
	dhFile, caCrtFile, authFile string,
) *ServerConfig {
	config := ServerConfig{NewConfig()}
	config.SetServerMode(1194, network, netmask)
	config.SetTlsCACertificate(caFile)
	config.SetTlsPrivatePubKeys(certFile, certKeyFile)
	config.SetTlsServer(dhFile, caCrtFile)
	config.SetTlsAuth(authFile)

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-CBC")
	config.setParam("verb", "3")
	config.setParam("tls-version-min", "1.2")
	config.setFlag("management-client-auth")
	config.setParam("verify-client-cert", "none")
	config.setParam("tls-cipher", "TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384")
	config.setParam("reneg-sec", "60")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	return &config
}

func NewClientConfig(
	remote string,
	caFile, authFile string,
) *ClientConfig {
	config := ClientConfig{NewConfig()}
	config.SetClientMode(remote, 1194)
	config.SetTlsCACertificate(caFile)
	config.SetTlsAuth(authFile)

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-CBC")
	config.setParam("verb", "3")
	config.setParam("tls-cipher", "TLS-ECDHE-RSA-WITH-AES-256-GCM-SHA384")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	config.setParam("reneg-sec", "60")
	config.setParam("resolv-retry", "infinite")
	config.setParam("setenv", "opt block-outside-dns")
	config.setParam("redirect-gateway", "def1 bypass-dhcp")
	config.setParam("dhcp-option", "DNS 208.67.222.222")
	config.setParam("dhcp-option", "DNS 208.67.220.220")

	return &config
}

func NewClientConfigFromString(configString, configFile string) (*ClientConfig, error) {
	err := ioutil.WriteFile(configFile, []byte(configString), 0600)
	if err != nil {
		return nil, err
	}

	config := ClientConfig{NewConfig()}
	config.AddOptions(OptionParam("config", configFile))
	return &config, nil
}
