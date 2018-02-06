package openvpn

type ServerConfig struct {
	*Config
}

func (c *ServerConfig) SetServerMode(port int, network, netmask string) {
	c.SetPort(port)
	c.setParam("server", network+" "+netmask)
	c.setParam("topology", "subnet")
}

func (c *ServerConfig) SetTlsServer(caCrtFile string) {
	c.setFlag("tls-server")
	c.AddOptions(OptionFile("crl-verify", caCrtFile))
}
