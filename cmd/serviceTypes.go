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

package cmd

import (
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	service_socks5 "github.com/mysteriumnetwork/node/services/socks5"
	socks5_service "github.com/mysteriumnetwork/node/services/socks5/service"
	service_wireguard "github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
)

var (
	serviceTypesRequestParser = map[string]endpoints.ServiceOptionsParser{
		service_noop.ServiceType:      service_noop.ParseJSONOptions,
		service_openvpn.ServiceType:   openvpn_service.ParseJSONOptions,
		service_wireguard.ServiceType: wireguard_service.ParseJSONOptions,
		service_socks5.ServiceType:    socks5_service.ParseJSONOptions,
	}
)
