/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package services

import (
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/pkg/errors"
)

// Types returns all possible service types
func Types() []string {
	return []string{openvpn.ServiceType, wireguard.ServiceType, noop.ServiceType}
}

// TypeConfiguredOptions returns specific service options
func TypeConfiguredOptions(serviceType string) (service.Options, error) {
	switch serviceType {
	case openvpn.ServiceType:
		return openvpn_service.GetOptions(), nil
	case wireguard.ServiceType:
		return wireguard_service.GetOptions(), nil
	case noop.ServiceType:
		return noop.GetOptions(), nil
	default:
		return nil, errors.Errorf("unknown service type: %q", serviceType)
	}
}
