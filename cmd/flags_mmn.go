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
	"gopkg.in/urfave/cli.v1"

	"github.com/mysteriumnetwork/node/core/node"
)

// RegisterFlagsDiscovery function register discovery flags to flag list
func RegisterFlagsMMN(flags *[]cli.Flag) {
	*flags = append(*flags, mmnAddressFlag)
}

// ParseFlagsDiscovery function fills in discovery options from CLI context
func ParseFlagsMMN(ctx *cli.Context) node.OptionsMMN {
	return node.OptionsMMN{
		Address: ctx.GlobalString(mmnAddressFlag.Name),
	}
}
