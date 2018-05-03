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

func (c *ServerConfig) SetProtocol(protocol string) {
	if protocol == "tcp" {
		c.setParam("proto", "tcp-server")
	} else if protocol == "udp" {
		c.setFlag("explicit-exit-notify")
	}
}
