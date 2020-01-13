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
	"github.com/urfave/cli/v2"
)

var (
	// FlagOpenvpnProtocol protocol for OpenVPN to use.
	FlagOpenvpnProtocol = cli.StringFlag{
		Name:  "openvpn.proto",
		Usage: "OpenVPN protocol to use. Options: { udp, tcp }",
		Value: "udp",
	}
	// FlagOpenvpnPort port for OpenVPN to use.
	FlagOpenvpnPort = cli.IntFlag{
		Name:  "openvpn.port",
		Usage: "OpenVPN port to use. If not specified, random port will be used",
		Value: 0,
	}
	// FlagOpenvpnSubnet OpenVPN subnet that will be used for connecting clients.
	FlagOpenvpnSubnet = cli.StringFlag{
		Name:  "openvpn.subnet",
		Usage: "OpenVPN subnet that will be used to connecting VPN clients",
		Value: "10.8.0.0",
	}
	// FlagOpenvpnNetmask OpenVPN subnet netmask.
	FlagOpenvpnNetmask = cli.StringFlag{
		Name:  "openvpn.netmask",
		Usage: "OpenVPN subnet netmask",
		Value: "255.255.255.0",
	}
)

// RegisterFlagsServiceOpenvpn registers OpenVPN CLI flags for parsing them later
func RegisterFlagsServiceOpenvpn(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagOpenvpnProtocol,
		&FlagOpenvpnPort,
		&FlagOpenvpnSubnet,
		&FlagOpenvpnNetmask,
	)
}

// ParseFlagsServiceOpenvpn parses CLI flags and registers value to configuration
func ParseFlagsServiceOpenvpn(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagOpenvpnProtocol)
	Current.ParseIntFlag(ctx, FlagOpenvpnPort)
	Current.ParseStringFlag(ctx, FlagOpenvpnSubnet)
	Current.ParseStringFlag(ctx, FlagOpenvpnNetmask)
}
