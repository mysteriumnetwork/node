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

package config

import (
	"path/filepath"
	"strconv"
)

// NewConfig creates new openvpn configuration structure and takes configuration directory as parameter for file param serialization
func NewConfig(runtimeDir string, scriptSearchPath string) *GenericConfig {
	return &GenericConfig{
		runtimeDir:       runtimeDir,
		scriptSearchPath: scriptSearchPath,
		options:          make([]configOption, 0),
	}
}

// GenericConfig represents openvpn configuration structure common for both client and server modes
type GenericConfig struct {
	runtimeDir       string
	scriptSearchPath string
	options          []configOption
}

type configOption interface {
	getName() string
}

// AddOptions adds a list of provided options to GenericConfig structure
func (c *GenericConfig) AddOptions(options ...configOption) {
	c.options = append(c.options, options...)
}

// SetParam add a named parameter to configuration with given value(s)
func (c *GenericConfig) SetParam(name string, values ...string) {
	c.AddOptions(
		OptionParam(name, values...),
	)
}

// SetFlag adds named "flag style" param (no values)
func (c *GenericConfig) SetFlag(name string) {
	c.AddOptions(
		OptionFlag(name),
	)
}

// SetManagementAddress creates TCP socket option for communication with openvpn process
func (c *GenericConfig) SetManagementAddress(ip string, port int) {
	c.SetParam("management", ip, strconv.Itoa(port))
	c.SetFlag("management-client")
}

// SetPort sets transport port for openvpn traffic
func (c *GenericConfig) SetPort(port int) {
	c.SetParam("port", strconv.Itoa(port))
}

// SetDevice sets device name for tun devices
func (c *GenericConfig) SetDevice(deviceName string) {
	c.SetParam("dev", deviceName)
}

// SetTLSCACertificate setups Certificate Authority parameter (in PEM format) for server certificate validation
func (c *GenericConfig) SetTLSCACertificate(caFile string) {
	c.AddOptions(OptionFile("ca", caFile, filepath.Join(c.runtimeDir, "ca.crt")))
}

// SetTLSPrivatePubKeys sets certificate and private key for TLS communication on server side
func (c *GenericConfig) SetTLSPrivatePubKeys(certFile string, certKeyFile string) {
	c.AddOptions(OptionFile("cert", certFile, filepath.Join(c.runtimeDir, "server.crt")))
	c.AddOptions(OptionFile("key", certKeyFile, filepath.Join(c.runtimeDir, "server.key")))
}

// SetTLSCrypt sets preshared TLS key on both client and server side
func (c *GenericConfig) SetTLSCrypt(cryptFile string) {
	c.AddOptions(OptionFile("tls-crypt", cryptFile, filepath.Join(c.runtimeDir, "ta.key")))
}

// SetReconnectRetry describes conditions which enforces client to close a session in case of failed authentication
func (c *GenericConfig) SetReconnectRetry(count int) {
	c.SetParam("connect-retry-max", strconv.Itoa(count))
	c.SetParam("remap-usr1", "SIGTERM")
	c.SetFlag("single-session")
	c.SetFlag("tls-exit")
}

// SetKeepAlive setups keepalive interval and timeout values
func (c *GenericConfig) SetKeepAlive(interval, timeout int) {
	c.SetParam("keepalive", strconv.Itoa(interval), strconv.Itoa(timeout))
}

// SetPingTimerRemote sets "ping from remote required" option
func (c *GenericConfig) SetPingTimerRemote() {
	c.SetFlag("ping-timer-rem")
}

// SetPersistTun sets persistent tunnel option for openvpn (i.e. do not remove tunnel on process exit)
func (c *GenericConfig) SetPersistTun() {
	c.SetFlag("persist-tun")
}

// SetPersistKey setups persted key option for openvpn
func (c *GenericConfig) SetPersistKey() {
	c.SetFlag("persist-key")
}

// SetScriptParam adds parameter with name and value which represents script path against script search directory
func (c *GenericConfig) SetScriptParam(paramName string, script Script) {
	fullPath := script.FullPath(c.scriptSearchPath)
	c.SetParam(paramName, fullPath)
}

// GetFullScriptPath returns full script path
func (c *GenericConfig) GetFullScriptPath(script Script) string {
	return script.FullPath(c.scriptSearchPath)
}
