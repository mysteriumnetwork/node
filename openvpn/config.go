package openvpn

import (
	"strconv"
	"strings"
)

type Config struct {
	flags  map[string]bool
	params []string
}

func NewConfig() *Config {
	return &Config{
		flags:  make(map[string]bool),
		params: make([]string, 0),
	}
}

func (c *Config) setParam(key, val string) {
	a := strings.Split("--"+key+" "+val, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

func (c *Config) setFlag(key string) {
	a := strings.Split("--"+key, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

func (c *Config) Validate() (config []string, err error) {
	return c.params, nil
}

func (c *Config) SetRemote(r string, port int) {
	c.setParam("port", strconv.Itoa(port))
	c.setParam("remote", r)
}

func (c *Config) SetDevice(t string) {
	c.setParam("dev", t)
}

func (c *Config) SetSecret(key string) {
	c.setParam("secret", key)
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

func (c *Config) SetManagementPath(path string) {
	c.setParam("management", path+" unix")
	c.setFlag("management-client")
	c.setFlag("management-hold")
	c.setFlag("management-signal")
	c.setFlag("management-up-down")
}