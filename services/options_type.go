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
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
)

// JSONParsersByType parsers of service specific options from JSON request.
var JSONParsersByType = map[string]ServiceOptionsParser{
	noop.ServiceType:         noop.ParseJSONOptions,
	openvpn.ServiceType:      openvpn_service.ParseJSONOptions,
	wireguard.ServiceType:    wireguard_service.ParseJSONOptions,
	scraping.ServiceType:     wireguard_service.ParseJSONOptions,
	datatransfer.ServiceType: wireguard_service.ParseJSONOptions,
	dvpn.ServiceType:         wireguard_service.ParseJSONOptions,
}

// ServiceOptionsParser parses request to service specific options
type ServiceOptionsParser func(*json.RawMessage) (service.Options, error)

// Types returns all possible service types.
func Types() []string {
	return []string{
		openvpn.ServiceType,
		wireguard.ServiceType,
		noop.ServiceType,
		scraping.ServiceType,
		datatransfer.ServiceType,
		dvpn.ServiceType,
	}
}

// TypeConfiguredOptions returns specific service options.
func TypeConfiguredOptions(serviceType string) (service.Options, error) {
	switch serviceType {
	case openvpn.ServiceType:
		return openvpn_service.GetOptions(), nil
	case wireguard.ServiceType:
		return wireguard_service.GetOptions(), nil
	case noop.ServiceType:
		return noop.GetOptions(), nil
	case scraping.ServiceType:
		return wireguard_service.GetOptions(), nil
	case datatransfer.ServiceType:
		return wireguard_service.GetOptions(), nil
	case dvpn.ServiceType:
		return wireguard_service.GetOptions(), nil
	default:
		return nil, errors.Errorf("unknown service type: %q", serviceType)
	}
}

// TypeJSONParser get parser to parse service specific options from JSON request.
func TypeJSONParser(serviceType string) (ServiceOptionsParser, error) {
	parser, exist := JSONParsersByType[serviceType]
	if !exist {
		return nil, errors.Errorf("unknown service type: %q", serviceType)
	}
	return parser, nil
}

// IsTypeValid returns true if a given string is valid service type.
func IsTypeValid(s string) bool {
	for _, v := range Types() {
		if v == s {
			return true
		}
	}
	return false
}
