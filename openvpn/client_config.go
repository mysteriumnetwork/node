package openvpn

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

type ClientConfig struct {
	*Config
}

func (c *ClientConfig) SetClientMode(serverIp string, serverPort int) {
	c.setFlag("client")
	c.setParam("remote", serverIp)
	c.SetPort(serverPort)
	c.setFlag("nobind")
	c.setParam("remote-cert-tls", "server")
}