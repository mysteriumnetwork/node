package openvpn

import (
	"strconv"
)

func NewConfig() *Config {
	return &Config{
		options:  make([]configOption, 0),
	}
}

type Config struct {
	options []configOption
}

type configOption interface {
	getName() string
}

func (c *Config) setParam(name, value string) {
	c.options = append(
		c.options,
		&optionParam{name, value},
	)
}

func (c *Config) setFlag(name string) {
	c.options = append(
		c.options,
		&optionFlag{name},
	)
}

func (c *Config) SetManagementPath(path string) {
	c.setParam("management", path+" unix")
	c.setFlag("management-signal")
	c.setFlag("management-up-down")
}

func (c *Config) SetPort(port int) {
	c.setParam("port", strconv.Itoa(port))
}

func (c *Config) SetDevice(deviceName string) {
	c.setParam("dev", deviceName)
}

func (c *Config) SetTlsCertificate(caFile, certFile, certKeyFile string) {
	c.setParam("ca", caFile)
	c.setParam("cert", certFile)
	c.setParam("key", certKeyFile)
}

func (c *Config) SetTlsAuth(authFile string) {
	c.setParam("tls-auth", authFile)
}

func (c *Config) SetKeepAlive(interval, timeout int) {
	c.setParam("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}

func (c *Config) SetPingTimerRemote() {
	c.setFlag("ping-timer-rem")
}

func (c *Config) SetPersistTun() {
	c.setFlag("persist-tun")
}

func (c *Config) SetPersistKey() {
	c.setFlag("persist-key")
}
