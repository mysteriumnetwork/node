/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
