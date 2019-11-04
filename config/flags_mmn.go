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
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
)

var (
	mmnAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "mymysterium.url",
		Usage: "URL of my.mysterium.network API",
		Value: metadata.DefaultNetwork.MMNAddress,
	})

	mmnEnabledFlag = altsrc.NewBoolTFlag(cli.BoolTFlag{
		Name:  "mymysterium.enabled",
		Usage: "Enables my.mysterium.network integration",
	})
)

// RegisterFlagsMMN function register mmn flags to flag list
func RegisterFlagsMMN(flags *[]cli.Flag) {
	*flags = append(*flags, mmnAddressFlag)
	*flags = append(*flags, mmnEnabledFlag)
}

// ParseFlagsMMN function fills in mmn options from CLI context
func ParseFlagsMMN(ctx *cli.Context) node.OptionsMMN {
	return node.OptionsMMN{
		Address: ctx.GlobalString(mmnAddressFlag.Name),
		Enabled: ctx.GlobalBoolT(mmnEnabledFlag.Name),
	}
}
