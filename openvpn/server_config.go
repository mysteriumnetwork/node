package openvpn

func NewServerConfig() *ServerConfig {
	config := ServerConfig{NewConfig()}

	return &config
}

type ServerConfig struct {
	*Config
}

func (c *ServerConfig) SetManagementPath(path string) {
	c.setParam("management", path+" unix")
	c.setFlag("management-server")
	c.setFlag("management-hold")
	c.setFlag("management-signal")
	c.setFlag("management-up-down")
}