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
	"github.com/urfave/cli"
)

var (
	uiPortFlag = cli.IntFlag{
		Name:  "ui.port",
		Usage: "the port to run ui on",
		Value: 4449,
	}
	uiEnableFlag = cli.BoolTFlag{
		Name:  "ui.enable",
		Usage: "enables the ui",
	}
)

// RegisterFlagsUI function register UI flags to flag list
func RegisterFlagsUI(flags *[]cli.Flag) {
	*flags = append(*flags, uiPortFlag, uiEnableFlag)
}

// ParseFlagsUI function fills in UI options from CLI context
func ParseFlagsUI(ctx *cli.Context) node.OptionsUI {
	return node.OptionsUI{
		UIEnabled: ctx.GlobalBool(uiEnableFlag.Name),
		UIPort:    ctx.GlobalInt(uiPortFlag.Name),
	}
}
