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

package service

import (
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/socks5"
	socks5_service "github.com/mysteriumnetwork/node/services/socks5/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/urfave/cli/v2"
)

var (
	serviceTypes = []string{"openvpn", "wireguard", "socks5", "noop"}

	serviceTypesFlagsParser = map[string]func(ctx *cli.Context) service.Options{
		noop.ServiceType: noop.ParseFlags,
		openvpn.ServiceType: func(ctx *cli.Context) service.Options {
			config.ParseFlagsServiceOpenvpn(ctx)
			return openvpn_service.GetOptions()
		},
		wireguard.ServiceType: func(ctx *cli.Context) service.Options {
			config.ParseFlagsServiceWireguard(ctx)
			return wireguard_service.GetOptions()
		},
		socks5.ServiceType: func(ctx *cli.Context) service.Options {
			config.ParseFlagsServiceSOCKS5(ctx)
			return socks5_service.GetOptions()
		},
	}
)
