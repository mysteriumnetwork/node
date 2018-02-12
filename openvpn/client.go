package openvpn

import "sync"
import "github.com/mysterium/node/openvpn/primitives"
import "github.com/mysterium/node/openvpn/management"

type Client interface {
	Start() error
	Wait() error
	Stop() error
}

type openVpnClient struct {
	config     *ClientConfig
	management *management.Management
	process    *Process
}

func NewClient(config *ClientConfig, directoryRuntime string, middlewares ...management.ManagementMiddleware) *openVpnClient {
	// Add the management interface socketAddress to the config
	socketAddress := tempFilename(directoryRuntime, "openvpn-management-", ".sock")
	config.SetManagementSocket(socketAddress)

	return &openVpnClient{
		config:     config,
		management: management.NewManagement(socketAddress, "[client-management] ", middlewares...),
		process:    NewProcess("[client-openvpn] "),
	}
}

// ConfigGenerator callback returns generated server config
type ClientConfigGenerator func() *ClientConfig

// NewServerConfigGenerator returns function generating server config and generates required security primitives
func NewClientConfigGenerator(directoryRuntime, vpnServerIP string) ClientConfigGenerator {
	return func() *ClientConfig {
		// (Re)generate required security primitives before openvpn start
		vpnClientConfig := NewClientConfig(
			vpnServerIP,
			primitives.CACertPath(directoryRuntime),
			primitives.TLSCryptKeyPath(directoryRuntime),
		)
		return vpnClientConfig
	}
}

func (client *openVpnClient) Start() error {
	// Start the management interface (if it isnt already started)
	err := client.management.Start()
	if err != nil {
		return err
	}

	// Fetch the current arguments
	arguments, err := ConfigToArguments(*client.config.Config)
	if err != nil {
		return err
	}

	return client.process.Start(arguments)
}

func (client *openVpnClient) Wait() error {
	return client.process.Wait()
}

func (client *openVpnClient) Stop() error {
	waiter := sync.WaitGroup{}

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		client.process.Stop()
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		client.management.Stop()
	}()

	waiter.Wait()
	return nil
}
