package openvpn

import (
	"io/ioutil"
)

const CLIENT_CONFIG_PATH = "client.ovpn"

func NewServerConfig(
	network, netmask string,
	caFile, certFile, certKeyFile,
	dhFile, caCrtFile, authFile string,
) *ServerConfig {
	config := ServerConfig{NewConfig()}
	config.SetServerMode(1194, network, netmask)
	config.SetTlsCertificate(caFile, certFile, certKeyFile)
	config.SetTlsServer(dhFile, caCrtFile)
	config.SetTlsAuth(authFile)

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-CBC")
	config.setParam("compress", "lz4")
	config.setParam("verb", "3")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	return &config
}

func NewClientConfig(
	remote string,
	caFile, certFile, certKeyFile, authFile string,
) *ClientConfig {
	config := ClientConfig{NewConfig()}
	config.SetClientMode(remote, 1194)
	config.SetTlsCertificate(caFile, certFile, certKeyFile)
	config.SetTlsAuth(authFile)

	config.SetDevice("tun")
	config.setParam("cipher", "AES-256-CBC")
	config.setParam("compress", "lz4")
	config.setParam("verb", "3")
	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	config.setParam("resolv-retry", "infinite")
	config.setParam("setenv", "opt block-outside-dns")
	config.setParam("redirect-gateway", "def1 bypass-dhcp")
	config.setParam("dhcp-option", "DNS 208.67.222.222")
	config.setParam("dhcp-option", "208.67.220.220")

	return &config
}

func NewClientConfigFromString(configString string) (*ClientConfig, error) {
	err := ioutil.WriteFile(CLIENT_CONFIG_PATH, []byte(configString), 0600)
	if err != nil {
		return nil, err
	}

	config := ClientConfig{NewConfig()}
	config.AddOptions(OptionParam("config", CLIENT_CONFIG_PATH))
	return &config, nil
}