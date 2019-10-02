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
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	"gopkg.in/urfave/cli.v1"
)

var (
	accountantAddressFlag = cli.StringFlag{
		Name:  "accountant.address",
		Usage: "accountant URL address",
		Value: metadata.DefaultNetwork.AccountantAddress,
	}
	accountantIDFlag = cli.StringFlag{
		Name:  "accountant.accountant-id",
		Usage: "accountant contract address used to register identity",
		Value: metadata.DefaultNetwork.AccountantID,
	}
)

// RegisterFlagsAccountant function register network flags to flag list
func RegisterFlagsAccountant(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		accountantAddressFlag,
		accountantIDFlag,
	)
}

// ParseFlagsAccountant function fills in accountant options from CLI context
func ParseFlagsAccountant(ctx *cli.Context) node.OptionsAccountant {
	return node.OptionsAccountant{
		AccountantEndpointAddress: ctx.GlobalString(accountantAddressFlag.Name),
		AccountantID:              ctx.GlobalString(accountantIDFlag.Name),
	}
}
