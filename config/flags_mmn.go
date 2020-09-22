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

var (
	// FlagMMNAddress URL Of my.mysterium.network API.
	FlagMMNAddress = cli.StringFlag{
		Name:  "mmn.url",
		Usage: "URL of my.mysterium.network API",
		Value: metadata.DefaultNetwork.MMNAddress,
	}
	// FlagMMNKey token Of my.mysterium.network API.
	FlagMMNKey = cli.StringFlag{
		Name:  "mmn.key",
		Usage: "Token of my.mysterium.network API",
		Value: "",
	}
)

// RegisterFlagsMMN function registers MMN flags to flag list.
func RegisterFlagsMMN(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagMMNAddress,
		&FlagMMNKey,
	)
}

// ParseFlagsMMN function fills in MMN options from CLI context.
func ParseFlagsMMN(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagMMNAddress)
	Current.ParseStringFlag(ctx, FlagMMNKey)
}
