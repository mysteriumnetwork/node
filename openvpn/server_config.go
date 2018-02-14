package openvpn

type ServerConfig struct {
	*Config
}

func (c *ServerConfig) SetServerMode(port int, network, netmask string) {
	c.SetPort(port)
	c.setParam("server", network+" "+netmask)
	c.setParam("topology", "subnet")
}

func (c *ServerConfig) SetTLSServer() {
	c.setFlag("tls-server")
	c.AddOptions(OptionFile("dh", "none"))
}
