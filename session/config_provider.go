package session

import (
	"github.com/mysteriumnetwork/node/openvpn"
)

// OpenVPNServiceConfigProvider is a service config provider for openvpn
type OpenVPNServiceConfigProvider struct {
	clientConfigGenerator openvpn.ClientConfigGenerator
}

// NewOpenVPNServiceConfigProvider creates a new instance of OpenVPNServiceConfigProvider
func NewOpenVPNServiceConfigProvider(generator openvpn.ClientConfigGenerator) ServiceConfigProvider {
	return OpenVPNServiceConfigProvider{
		clientConfigGenerator: generator,
	}
}

// ProvideServiceConfig callback providing service configuration for a session
func (provider OpenVPNServiceConfigProvider) ProvideServiceConfig() (ServiceConfiguration, error) {
	return provider.clientConfigGenerator(), nil
}
