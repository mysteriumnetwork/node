/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package openvpn

import (
	"path/filepath"
	"strconv"
)

// NewConfig creates new openvpn configuration structure and takes configuration directory as parameter for file param serialization
func NewConfig(configDir string) *Config {
	return &Config{
		configDir: configDir,
		options:   make([]configOption, 0),
	}
}

// Config represents openvpn configuration structure
type Config struct {
	configDir string
	options   []configOption
}

type configOption interface {
	getName() string
}

// AddOptions adds a list of provided options to Config structure
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

// SetManagementSocket creates unix socket style socket option for communication with openvpn process
func (c *Config) SetManagementSocket(socketAddress string) {
	c.setParam("management", socketAddress+" unix")
	c.setFlag("management-client")
}

// SetPort sets transport port for openvpn traffic
func (c *Config) SetPort(port int) {
	c.setParam("port", strconv.Itoa(port))
}

// SetDevice sets device name for tun devices
func (c *Config) SetDevice(deviceName string) {
	c.setParam("dev", deviceName)
}

// SetTLSCACertificate setups Certificate Authority parameter (in PEM format) for server certificate validation
func (c *Config) SetTLSCACertificate(caFile string) {
	c.AddOptions(OptionFile("ca", caFile, filepath.Join(c.configDir, "ca.crt")))
}

// SetTLSPrivatePubKeys sets certificate and private key for TLS communication on server side
func (c *Config) SetTLSPrivatePubKeys(certFile string, certKeyFile string) {
	c.AddOptions(OptionFile("cert", certFile, filepath.Join(c.configDir, "server.crt")))
	c.AddOptions(OptionFile("key", certKeyFile, filepath.Join(c.configDir, "server.key")))
}

// SetTLSCrypt sets preshared TLS key on both client and server side
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

// SetKeepAlive setups keepalive interval and timeout values
func (c *Config) SetKeepAlive(interval, timeout int) {
	c.setParam("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}

// SetPingTimerRemote sets "ping from remote required" option
func (c *Config) SetPingTimerRemote() {
	c.setFlag("ping-timer-rem")
}

// SetPersistTun sets persistent tunnel option for openvpn (i.e. do not remove tunnel on process exit)
func (c *Config) SetPersistTun() {
	c.setFlag("persist-tun")
}

// SetPersistKey setups persted key option for openvpn
func (c *Config) SetPersistKey() {
	c.setFlag("persist-key")
}
