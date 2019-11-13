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
	"gopkg.in/urfave/cli.v1"
)

var (
	// FlagTransactorAddress transactor URL.
	FlagTransactorAddress = cli.StringFlag{
		Name:  "transactor.address",
		Usage: "Transactor URL address",
		Value: metadata.DefaultNetwork.TransactorAddress,
	}
	// FlagTransactorRegistryAddress registry contract address used for identity registration.
	FlagTransactorRegistryAddress = cli.StringFlag{
		Name:  "transactor.registry-address",
		Usage: "Registry contract address used to register identity",
		Value: metadata.DefaultNetwork.RegistryAddress,
	}
	// FlagTransactorAccountantID accountant contract address used for identity registration.
	FlagTransactorAccountantID = cli.StringFlag{
		Name:  "transactor.accountant-id",
		Usage: "Accountant contract address used to register identity",
		Value: metadata.DefaultNetwork.AccountantID,
	}
)

// RegisterFlagsTransactor function register network flags to flag list
func RegisterFlagsTransactor(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		FlagTransactorAddress,
		FlagTransactorRegistryAddress,
		FlagTransactorAccountantID,
	)
}

// ParseFlagsTransactor function fills in transactor options from CLI context
func ParseFlagsTransactor(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagTransactorAddress)
	Current.ParseStringFlag(ctx, FlagTransactorRegistryAddress)
	Current.ParseStringFlag(ctx, FlagTransactorAccountantID)
}
