/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	openvpn_core "github.com/mysteriumnetwork/go-openvpn/openvpn/core"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/urfave/cli"
)

var (
	tequilapiAddressFlag = cli.StringFlag{
		Name:  "tequilapi.address",
		Usage: "IP address of interface to listen for incoming connections",
		Value: "127.0.0.1",
	}
	tequilapiPortFlag = cli.IntFlag{
		Name:  "tequilapi.port",
		Usage: "Port for listening incoming api requests",
		Value: 4050,
	}
)

// RegisterFlagsNode function register node flags to flag list
func RegisterFlagsNode(flags *[]cli.Flag) error {
	if err := RegisterFlagsDirectory(flags); err != nil {
		return err
	}

	*flags = append(*flags, tequilapiAddressFlag, tequilapiPortFlag)

	RegisterFlagsNetwork(flags)
	openvpn_core.RegisterFlags(flags)
	RegisterFlagsLocation(flags)

	return nil
}

// ParseFlagsNode function fills in node options from CLI context
func ParseFlagsNode(ctx *cli.Context) node.Options {
	return node.Options{
		ParseFlagsDirectory(ctx),

		ctx.GlobalString(tequilapiAddressFlag.Name),
		ctx.GlobalInt(tequilapiPortFlag.Name),

		openvpn_core.ParseFlags(ctx),
		ParseFlagsLocation(ctx),
		ParseFlagsNetwork(ctx),
	}
}
