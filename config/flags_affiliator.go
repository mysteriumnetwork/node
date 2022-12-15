/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/metadata"
)

var (
	// FlagAffiliatorAddress affiliator URL.
	FlagAffiliatorAddress = cli.StringFlag{
		Name:  metadata.FlagNames.AffiliatorAddress,
		Usage: "Affiliator URL address",
		Value: metadata.DefaultNetwork.AffiliatorAddress,
	}
)

// RegisterFlagsAffiliator function register network flags to flag list
func RegisterFlagsAffiliator(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagAffiliatorAddress,
	)
}

// ParseFlagsAffiliator function fills in affiliator options from CLI context
func ParseFlagsAffiliator(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagAffiliatorAddress)
}
