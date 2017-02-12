package openvpn

import (
	"strings"
)

func NewConfig() *Config {
	return &Config{
		flags:  make(map[string]bool),
		params: make([]string, 0),
	}
}

type Config struct {
	flags  map[string]bool
	params []string
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