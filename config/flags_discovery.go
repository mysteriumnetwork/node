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

package config

import (
	"fmt"

	"github.com/mysteriumnetwork/node/core/node"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

var (
	discoveryTypeFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "discovery.type",
		Usage: fmt.Sprintf("Proposal discovery adapter. Options: { %s, %s }", node.DiscoveryTypeAPI, node.DiscoveryTypeBroker),
		Value: string(node.DiscoveryTypeAPI),
	})
	discoveryAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name: "discovery.address",
		Usage: fmt.Sprintf(
			"Address of specific discovery adapter given in '--%s'",
			discoveryTypeFlag.Name,
		),
		Value: apiAddressFlag.Value,
	})
)

// RegisterFlagsDiscovery function register discovery flags to flag list
func RegisterFlagsDiscovery(flags *[]cli.Flag) {
	*flags = append(*flags, discoveryTypeFlag, discoveryAddressFlag)
}

// ParseFlagsDiscovery function fills in discovery options from CLI context
func ParseFlagsDiscovery(ctx *cli.Context) node.OptionsDiscovery {
	return node.OptionsDiscovery{
		Type:    node.DiscoveryType(ctx.GlobalString(discoveryTypeFlag.Name)),
		Address: ctx.GlobalString(discoveryAddressFlag.Name),
	}
}
