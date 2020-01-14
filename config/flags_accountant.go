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
	// FlagAccountantAddress points to the accountant service
	FlagAccountantAddress = cli.StringFlag{
		Name:  "accountant.address",
		Usage: "accountant URL address",
		Value: metadata.DefaultNetwork.AccountantAddress,
	}
	// FlagAccountantID determines the accountant ID
	FlagAccountantID = cli.StringFlag{
		Name:  "accountant.accountant-id",
		Usage: "accountant contract address used to register identity",
		Value: metadata.DefaultNetwork.AccountantID,
	}
)

// RegisterFlagsAccountant function register network flags to flag list
func RegisterFlagsAccountant(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagAccountantAddress,
		&FlagAccountantID,
	)
}

// ParseFlagsAccountant function fills in accountant options from CLI context
func ParseFlagsAccountant(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagAccountantAddress)
	Current.ParseStringFlag(ctx, FlagAccountantID)
}
