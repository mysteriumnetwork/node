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

package cmd

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/core/service"
	service_noop "github.com/mysteriumnetwork/node/services/noop"
	service_openvpn "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	service_wireguard "github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/urfave/cli"
)

var (
	serviceTypesAvailable = []string{"openvpn", "wireguard", "noop"}
	serviceTypesEnabled   = []string{"openvpn", "noop"}

	serviceTypesFlagsParser = map[string]func(ctx *cli.Context) service.Options{
		service_noop.ServiceType:      service_noop.ParseCLIFlags,
		service_openvpn.ServiceType:   openvpn_service.ParseCLIFlags,
		service_wireguard.ServiceType: wireguard_service.ParseCLIFlags,
	}

	serviceTypesRequestParser = map[string]func(request json.RawMessage) (service.Options, error){
		service_noop.ServiceType:      service_noop.ParseJSONOptions,
		service_openvpn.ServiceType:   openvpn_service.ParseJSONOptions,
		service_wireguard.ServiceType: wireguard_service.ParseJSONOptions,
	}
)
