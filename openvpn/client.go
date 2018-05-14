package openvpn

import "sync"
import (
	"github.com/mysterium/node/openvpn/management"
	"github.com/mysterium/node/openvpn/tls"
	"github.com/mysterium/node/session"
)

// Client defines client process interfaces with basic commands
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

// NewClient creates openvpn client with given config params
func NewClient(openvpnBinary string, config *ClientConfig, directoryRuntime string, middlewares ...management.Middleware) *openVpnClient {
	// Add the management interface socketAddress to the config
	socketAddress := tempFilename(directoryRuntime, "openvpn-management-", ".sock")
	config.SetManagementSocket(socketAddress)

	return &openVpnClient{
		config:     config,
		management: management.NewManagement(socketAddress, "[client-management] ", middlewares...),
		process:    NewProcess(openvpnBinary, "[client-openvpn] "),
	}
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}

// ClientConfigGenerator callback returns generated server config
type ClientConfigGenerator func() *VPNConfig

func (generator ClientConfigGenerator) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return generator(), nil
}

// NewClientConfigGenerator returns function generating config params for remote client
func NewClientConfigGenerator(primitives *tls.Primitives, vpnServerIP string, port int, protocol string) ClientConfigGenerator {
	return func() *VPNConfig {
		return &VPNConfig{
			vpnServerIP,
			port,
			protocol,
			primitives.PresharedKey.ToPEMFormat(),
			primitives.CertificateAuthority.ToPEMFormat(),
		}
	}
}

func (client *openVpnClient) Start() error {
	// Start the management interface (if it isnt already started)
	err := client.management.Start()
	if err != nil {
		return err
	}

	// Fetch the current arguments
	arguments, err := (*client.config).ConfigToArguments()
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
