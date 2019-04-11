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

package cmd

import (
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/urfave/cli"
)

var (
	ipDetectorURLFlag = cli.StringFlag{
		Name:  "ip-detector",
		Usage: "Address (URL form) of ip detection service",
		Value: "https://testnet-location.mysterium.network/api/v1/location",
	}

	locationTypeFlag = cli.StringFlag{
		Name:  "location.type",
		Usage: "Service location detection type",
		Value: "builtin",
	}
	locationAddressFlag = cli.StringFlag{
		Name:  "location.address",
		Usage: "Address of the service location system",
		Value: "https://testnet-location.mysterium.network/api/v1/location",
	}
	locationCountryFlag = cli.StringFlag{
		Name:  "location.country",
		Usage: "Service location country. If not given country is autodetected",
		Value: "",
	}
	locationCityFlag = cli.StringFlag{
		Name:  "location.city",
		Usage: "Service location city",
		Value: "",
	}
	locationNodeTypeFlag = cli.StringFlag{
		Name:  "location.node-type",
		Usage: "Service location node type",
		Value: "",
	}
)

// RegisterFlagsLocation function register location flags to flag list
func RegisterFlagsLocation(flags *[]cli.Flag) {
	*flags = append(*flags, ipDetectorURLFlag,
		locationTypeFlag, locationAddressFlag, locationCountryFlag, locationCityFlag, locationNodeTypeFlag)
}

// ParseFlagsLocation function fills in location options from CLI context
func ParseFlagsLocation(ctx *cli.Context) node.OptionsLocation {
	return node.OptionsLocation{
		IPDetectorURL: ctx.GlobalString(ipDetectorURLFlag.Name),

		Type:     ctx.GlobalString(locationTypeFlag.Name),
		Address:  ctx.GlobalString(locationAddressFlag.Name),
		Country:  ctx.GlobalString(locationCountryFlag.Name),
		City:     ctx.GlobalString(locationCityFlag.Name),
		NodeType: ctx.GlobalString(locationNodeTypeFlag.Name),
	}
}
