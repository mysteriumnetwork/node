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
	"github.com/mysterium/node/openvpn/management"

	"errors"
	"github.com/mysterium/node/openvpn/tls"
	"sync"
	"time"
)

// NewServer constructs new openvpn server instance
func NewServer(openvpnBinary string, generateConfig ServerConfigGenerator, middlewares ...management.Middleware) *Server {

	return &Server{
		generateConfig: generateConfig,
		management:     management.NewManagement(management.LocalhostOnRandomPort, "[server-management] ", middlewares...),
		process:        NewProcess(openvpnBinary, "[server-openvpn] "),
	}
}

// ServerConfigGenerator callback returns generated server config
type ServerConfigGenerator func() *ServerConfig

// NewServerConfigGenerator returns function generating server config and generates required security primitives
func NewServerConfigGenerator(directoryRuntime string, primitives *tls.Primitives, port int, protocol string) ServerConfigGenerator {
	return func() *ServerConfig {
		vpnServerConfig := NewServerConfig(
			directoryRuntime,
			"10.8.0.0", "255.255.255.0",
			primitives,
			port,
			protocol,
		)
		return vpnServerConfig
	}
}

// Server structure describes openvpn server
type Server struct {
	generateConfig ServerConfigGenerator
	management     *management.Management
	process        *Process
}

// Start starts openvpn server generating required config and starting management interface on the way
func (server *Server) Start() error {
	config := server.generateConfig()
	addr, connected, err := server.management.WaitForConnection()
	if err != nil {
		return err
	}
	config.SetManagementAddress(addr.IP, addr.Port)

	// Fetch the current params
	arguments, err := (*config).ConfigToArguments()
	if err != nil {
		return err
	}

	//nil returned from process.Start doesn't guarantee that openvpn itself initialized correctly and accepted all arguments
	//it simply means that OS started process with specified args
	err = server.process.Start(arguments)
	if err != nil {
		server.management.Stop()
		return err
	}

	select {
	case connAccepted := <-connected:
		if connAccepted {
			return nil
		}
		return errors.New("management failed to accept connection")
	case <-time.After(2 * time.Second):
		return errors.New("management connection wait timeout")
	}

}

// Wait waits for openvpn server to exit
func (server *Server) Wait() error {
	return server.process.Wait()
}

// Stop instructs openvpn server to stop
func (server *Server) Stop() {
	waiter := sync.WaitGroup{}

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		server.process.Stop()
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		server.management.Stop()
	}()

	waiter.Wait()
}
