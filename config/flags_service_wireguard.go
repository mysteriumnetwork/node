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
	// FlagWireguardListenPorts range of listen ports.
	FlagWireguardListenPorts = cli.StringFlag{
		Name:  "wireguard.listen.ports",
		Usage: "Range of listen ports (e.g. 52820:53075), value of 0:0 means disabled",
		Value: "0:0",
	}
	// FlagWireguardListenSubnet subnet to be used by the wireguard service.
	FlagWireguardListenSubnet = cli.StringFlag{
		Name:  "wireguard.allowed.subnet",
		Usage: "Subnet to be used by the wireguard service",
		Value: "10.182.0.0/16",
	}
	// FlagWireguardPriceMinute sets the price per minute for provided wireguard service.
	FlagWireguardPriceMinute = cli.Float64Flag{
		Name:  "wireguard.price-minute",
		Usage: "Sets the price of the wireguard service per minute.",
	}
	// FlagWireguardPriceGB sets the price per GiB for provided wireguard service.
	FlagWireguardPriceGB = cli.Float64Flag{
		Name:  "wireguard.price-gb",
		Usage: "Sets the price of the wireguard service per minute.",
	}
	// FlagWireguardAccessPolicies a comma-separated list of access policies that determines allowed identities to use the service.
	FlagWireguardAccessPolicies = cli.StringFlag{
		Name:  "wireguard.access-policies",
		Usage: "Comma separated list that determines the access policies of the wireguard service.",
	}
)

// RegisterFlagsServiceWireguard function register Wireguard flags to flag list
func RegisterFlagsServiceWireguard(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagWireguardListenPorts,
		&FlagWireguardListenSubnet,
		&FlagWireguardPriceMinute,
		&FlagWireguardPriceGB,
		&FlagWireguardAccessPolicies,
	)
}

// ParseFlagsServiceWireguard parses CLI flags and registers value to configuration
func ParseFlagsServiceWireguard(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagWireguardListenPorts)
	Current.ParseStringFlag(ctx, FlagWireguardListenSubnet)
	Current.ParseFloat64Flag(ctx, FlagWireguardPriceMinute)
	Current.ParseFloat64Flag(ctx, FlagWireguardPriceGB)
	Current.ParseStringFlag(ctx, FlagWireguardAccessPolicies)
}
