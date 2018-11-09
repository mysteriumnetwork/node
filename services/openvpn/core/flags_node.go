/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package core

import (
	"github.com/urfave/cli"
)

var (
	binaryFlag = cli.StringFlag{
		Name:  "openvpn.binary",
		Usage: "openvpn binary to use for Open VPN connections",
		Value: "openvpn",
	}
)

// RegisterFlags function register Openvpn flags to flag list
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, binaryFlag)
}

// ParseFlags function fills in Openvpn options from CLI context
func ParseFlags(ctx *cli.Context) NodeOptions {
	return NodeOptions{
		ctx.GlobalString(binaryFlag.Name),
	}
}
