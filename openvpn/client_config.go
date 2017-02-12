package openvpn

import (
	"strconv"
)

func NewClientConfig(remote string, key string) *ClientConfig {
	config := ClientConfig{NewConfig()}
	config.SetRemote(remote, 1194)
	config.SetDevice("tun")
	config.SetSecret(key)

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
	c.setParam("port", strconv.Itoa(port))
	c.setParam("remote", r)
}

func (c *ClientConfig) SetDevice(t string) {
	c.setParam("dev", t)
}

func (c *ClientConfig) SetSecret(key string) {
	c.setParam("secret", key)
}

func (c *ClientConfig) SetKeepAlive(interval, timeout int) {
	c.setParam("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}

func (c *ClientConfig) SetPingTimerRemote() {
	c.setFlag("ping-timer-rem")
}

func (c *ClientConfig) SetPersistTun() {
	c.setFlag("persist-tun")
}

func (c *ClientConfig) SetPersistKey() {
	c.setFlag("persist-key")
}

func (c *ClientConfig) SetManagementPath(path string) {
	c.setParam("management", path+" unix")
	c.setFlag("management-client")
	c.setFlag("management-hold")
	c.setFlag("management-signal")
	c.setFlag("management-up-down")
}