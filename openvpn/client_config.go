package openvpn

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
