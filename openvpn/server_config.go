package openvpn

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
	c.AddOptions(OptionFile("dh", dhFile))
	c.AddOptions(OptionFile("crl-verify", caCrtFile))

}