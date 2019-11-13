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
)

var (
	// FlagWireguardConnectDelay consumer is delayed by the specified time if provider is behind the NAT.
	FlagWireguardConnectDelay = cli.IntFlag{
		Name:  "wireguard.connect.delay",
		Usage: "Consumer is delayed by specified time if provider is behind NAT",
		Value: 2000,
	}
	// FlagWireguardListenPorts range of listen ports.
	FlagWireguardListenPorts = cli.StringFlag{
		Name:  "wireguard.listen.ports",
		Usage: "Range of listen ports (e.g. 52820:53075)",
		Value: "0:0",
	}
	// FlagWireguardListenSubnet subnet to be used by the wireguard service.
	FlagWireguardListenSubnet = cli.StringFlag{
		Name:  "wireguard.allowed.subnet",
		Usage: "Subnet to be used by the wireguard service",
		Value: "10.182.0.0/16",
	}
)

// RegisterFlagsServiceWireguard function register Wireguard flags to flag list
func RegisterFlagsServiceWireguard(flags *[]cli.Flag) {
	*flags = append(*flags,
		FlagWireguardConnectDelay,
		FlagWireguardListenPorts,
		FlagWireguardListenSubnet,
	)
}

// ParseFlagsServiceWireguard parses CLI flags and registers value to configuration
func ParseFlagsServiceWireguard(ctx *cli.Context) {
	Current.ParseIntFlag(ctx, FlagWireguardConnectDelay)
	Current.ParseStringFlag(ctx, FlagWireguardListenPorts)
	Current.ParseStringFlag(ctx, FlagWireguardListenSubnet)
}
