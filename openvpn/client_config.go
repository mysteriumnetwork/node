package openvpn

import (
	"strconv"
)

func NewClientConfig(remote string, secretKey string) *ClientConfig {
	config := ClientConfig{NewConfig()}
	config.SetRemote(remote, 1194)
	config.SetDevice("tun")
	config.SetSecretKey(secretKey)

	config.SetKeepAlive(10, 60)
	config.SetPingTimerRemote()
	config.SetPersistTun()
	config.SetPersistKey()

	return &config
}

type ClientConfig struct {
	*Config
}

func (c *ClientConfig) SetRemote(r string, port int) {
	c.setParam("remote", r)
	c.setParam("port", strconv.Itoa(port))
}