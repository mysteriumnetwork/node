package openvpn

type ClientConfig struct {
	*Config
}

func (c *ClientConfig) SetClientMode(serverIP string, serverPort int) {
	c.setFlag("client")
	c.setParam("script-security", "2")
	c.setFlag("auth-nocache")
	c.setParam("remote", serverIP)
	c.SetPort(serverPort)
	c.setFlag("nobind")
	c.setParam("remote-cert-tls", "server")
	c.setFlag("auth-user-pass")
	c.setFlag("management-query-passwords")
}

func (c *ClientConfig) SetProtocol(protocol string) {
	if protocol == "tcp" {
		c.setParam("proto", "tcp-client")
	} else if protocol == "udp" {
		c.setFlag("explicit-exit-notify")
	}
}
