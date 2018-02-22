package openvpn

import (
	"github.com/mysterium/node/openvpn/management"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn/primitives"
	"github.com/mysterium/node/service_discovery/dto"
	"sync"
)

// NewServer constructs new openvpn server instance
func NewServer(openvpnBinary string, generateConfig ServerConfigGenerator, directoryRuntime string, middlewares ...management.Middleware) *Server {
	// Add the management interface socketAddress to the config
	socketAddress := tempFilename(directoryRuntime, "openvpn-management-", ".sock")
	return &Server{
		generateConfig: generateConfig,
		management:     management.NewManagement(socketAddress, "[server-management] ", middlewares...),
		process:        NewProcess(openvpnBinary, "[server-openvpn] "),
	}
}

// ConfigGenerator callback returns generated server config
type ServerConfigGenerator func() (*ServerConfig, error)

// NewServerConfigGenerator returns function generating server config and generates required security primitives
func NewServerConfigGenerator(directoryRuntime string, serviceLocation dto.Location, providerID identity.Identity) ServerConfigGenerator {
	return func() (*ServerConfig, error) {
		// (Re)generate required security primitives before openvpn start
		openVPNPrimitives, err := primitives.GenerateOpenVPNSecPrimitives(directoryRuntime, serviceLocation, providerID)
		if err != nil {
			return nil, err
		}
		vpnServerConfig := NewServerConfig(
			"10.8.0.0", "255.255.255.0",
			openVPNPrimitives,
		)
		return vpnServerConfig, err
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
	config, err := server.generateConfig()
	if err != nil {
		return err
	}

	config.SetManagementSocket(server.management.SocketAddress())

	// Start the management interface (if it isnt already started)
	if err := server.management.Start(); err != nil {
		return err
	}

	// Fetch the current params
	arguments, err := ConfigToArguments(*config.Config)
	if err != nil {
		return err
	}

	return server.process.Start(arguments)
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
