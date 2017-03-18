package openvpn

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

type ServerConfig struct {
	*Config
}

func (c *ServerConfig) SetServerMode(port int, network, netmask string) {
	c.SetPort(port)
	c.setParam("server", network + " " + netmask)
	c.setParam("topology", "subnet")
}

func (c *ServerConfig) SetTlsServer(dhFile, caCrtFile string) {
	c.setFlag("tls-server")
	c.setParam("dh", dhFile)
	c.setParam("crl-verify", caCrtFile)

}