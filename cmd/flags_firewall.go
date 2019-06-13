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
	enableKillSwitch = cli.BoolFlag{
		Name:  "firewall.killSwitch",
		Usage: "Enable consumer outgoing non tunneled traffic blocking during connections",
	}
	alwaysBlock = cli.BoolFlag{
		Name:  "firewall.killSwitch.always",
		Usage: "Always block non-tunneled outgoing consumer traffic",
	}
)

// RegisterFirewallFlags registers flags to control firewall killswitch
func RegisterFirewallFlags(flags *[]cli.Flag) {
	*flags = append(*flags, enableKillSwitch, alwaysBlock)
}

// ParseFirewallFlags parses registered flags and puts them into options structure
func ParseFirewallFlags(ctx *cli.Context) node.OptionsFirewall {
	return node.OptionsFirewall{
		EnableKillSwitch: ctx.GlobalBool(enableKillSwitch.Name),
		BlockAlways:      ctx.GlobalBool(alwaysBlock.Name),
	}
}
