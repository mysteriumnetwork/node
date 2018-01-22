package openvpn

type ClientConfig struct {
	*Config
}

func (c *ClientConfig) SetClientMode(serverIP string, serverPort int) {
	c.setFlag("client")
	c.setParam("remote", serverIP)
	c.SetPort(serverPort)
	c.setFlag("nobind")
	c.setParam("remote-cert-tls", "server")
}
