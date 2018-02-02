package openvpn

import (
	"path/filepath"
	"strconv"
)

func NewConfig() *Config {
	return &Config{
		options: make([]configOption, 0),
	}
}

type Config struct {
	options []configOption
}

type configOption interface {
	getName() string
}

func (c *Config) AddOptions(options ...configOption) {
	c.options = append(c.options, options...)
}

func (c *Config) setParam(name, value string) {
	c.AddOptions(
		OptionParam(name, value),
	)
}

func (c *Config) setFlag(name string) {
	c.AddOptions(
		OptionFlag(name),
	)
}

func (c *Config) SetManagementSocket(socketAddress string) {
	c.setParam("management", socketAddress+" unix")
	c.setFlag("management-client")
}

func (c *Config) SetScripts() {
	c.setParam("up", "update-resolv-conf")
	c.setParam("down", "update-resolv-conf")
}

func (c *Config) SetPort(port int) {
	c.setParam("port", strconv.Itoa(port))
}

func (c *Config) SetDevice(deviceName string) {
	c.setParam("dev", deviceName)
}

func (c *Config) SetTLSCACertificate(caFile string) {
	c.AddOptions(OptionFile("ca", caFile))
}

func (c *Config) SetTLSPrivatePubKeys(certFile string, certKeyFile string) {
	c.AddOptions(OptionFile("cert", certFile))
	c.AddOptions(OptionFile("key", certKeyFile))
}

func (c *Config) SetTlsAuth(authFile string) {
	c.AddOptions(OptionFile("tls-auth", authFile))
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
