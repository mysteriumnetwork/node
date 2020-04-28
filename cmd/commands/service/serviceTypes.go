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
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/urfave/cli/v2"
)

var (
	serviceTypes = []string{"openvpn", "wireguard", "noop"}

	serviceTypesFlagsParser = map[string]func(ctx *cli.Context) (service.Options, contract.PaymentMethodDTO){
		noop.ServiceType: func(ctx *cli.Context) (service.Options, contract.PaymentMethodDTO) {
			config.ParseFlagsServiceNoop(ctx)
			payment := contract.NewPaymentMethodDTO(pingpong.NewPaymentMethod(
				0,
				config.GetFloat64(config.FlagNoopPriceMinute),
			))
			return noop.GetOptions(), payment
		},
		openvpn.ServiceType: func(ctx *cli.Context) (service.Options, contract.PaymentMethodDTO) {
			config.ParseFlagsServiceOpenvpn(ctx)
			payment := contract.NewPaymentMethodDTO(pingpong.NewPaymentMethod(
				config.GetFloat64(config.FlagOpenVPNPriceGB),
				config.GetFloat64(config.FlagOpenVPNPriceMinute),
			))
			return openvpn_service.GetOptions(), payment
		},
		wireguard.ServiceType: func(ctx *cli.Context) (service.Options, contract.PaymentMethodDTO) {
			config.ParseFlagsServiceWireguard(ctx)
			payment := contract.NewPaymentMethodDTO(pingpong.NewPaymentMethod(
				config.GetFloat64(config.FlagWireguardPriceGB),
				config.GetFloat64(config.FlagWireguardPriceMinute),
			))
			return wireguard_service.GetOptions(), payment
		},
	}
)
