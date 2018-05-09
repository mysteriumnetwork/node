package openvpn

import (
	"path/filepath"
	"strconv"
)

func NewConfig(configdir string) *Config {
	return &Config{
		configDir: configdir,
		options:   make([]configOption, 0),
	}
}

type Config struct {
	configDir string
	options   []configOption
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

func (c *Config) SetPort(port int) {
	c.setParam("port", strconv.Itoa(port))
}

func (c *Config) SetDevice(deviceName string) {
	c.setParam("dev", deviceName)
}

func (c *Config) SetTLSCACertificate(caFile string) {
	c.AddOptions(OptionFile("ca", caFile, filepath.Join(c.configDir, "ca.crt")))
}

func (c *Config) SetTLSPrivatePubKeys(certFile string, certKeyFile string) {
	c.AddOptions(OptionFile("cert", certFile, filepath.Join(c.configDir, "server.crt")))
	c.AddOptions(OptionFile("key", certKeyFile, filepath.Join(c.configDir, "server.key")))
}

func (c *Config) SetTLSCrypt(cryptFile string) {
	c.AddOptions(OptionFile("tls-crypt", cryptFile, filepath.Join(c.configDir, "ta.key")))
}

// RestrictReconnects describes conditions which enforces client to close a session in case of failed authentication
func (c *Config) RestrictReconnects() {
	c.setParam("connect-retry-max", "2")
	c.setParam("remap-usr1", "SIGTERM")
	c.setFlag("single-session")
	c.setFlag("tls-exit")
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
