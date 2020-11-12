/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

// FlagPilvytisAddress address of Pilvytis service.
var FlagPilvytisAddress = cli.StringFlag{
	Name:  "pilvytis.address",
	Usage: "full address of the pilvytis service",
	Value: metadata.DefaultNetwork.PilvytisAddress,
}

// RegisterFlagsPilvytis func registers pilvytis flags to flag list.
func RegisterFlagsPilvytis(flags *[]cli.Flag) {
	*flags = append(*flags, &FlagPilvytisAddress)
}

// ParseFlagPilvytis func fills the pilvytis options from CLI context.
func ParseFlagPilvytis(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagPilvytisAddress)
}
