package openvpn

func NewServerConfig(secretKey string) *ServerConfig {
	config := ServerConfig{NewConfig()}
	config.SetDevice("tun")
	config.SetSecretKey(secretKey)

	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	return &config
}

type ServerConfig struct {
	*Config
}