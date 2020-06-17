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
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

var (
	// FlagHermesAddress points to the hermes service
	FlagHermesAddress = cli.StringFlag{
		Name:  "hermes.address",
		Usage: "hermes URL address",
		Value: metadata.DefaultNetwork.HermesAddress,
	}
	// FlagHermesID determines the hermes ID
	FlagHermesID = cli.StringFlag{
		Name:  "hermes.hermes-id",
		Usage: "hermes contract address used to register identity",
		Value: metadata.DefaultNetwork.HermesID,
	}
)

// RegisterFlagsHermes function register network flags to flag list
func RegisterFlagsHermes(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagHermesAddress,
		&FlagHermesID,
	)
}

// ParseFlagsHermes function fills in hermes options from CLI context
func ParseFlagsHermes(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagHermesAddress)
	Current.ParseStringFlag(ctx, FlagHermesID)
}
