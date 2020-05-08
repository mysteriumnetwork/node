/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	// FlagNoopPriceMinute sets the price per minute for provided noop service.
	FlagNoopPriceMinute = cli.Float64Flag{
		Name:   "noop.price-minute",
		Usage:  "Sets the price of the noop service per minute.",
		Hidden: true,
	}
	// FlagNoopPriceGB sets the price per GiB for provided OpenVPN service.
	FlagNoopPriceGB = cli.Float64Flag{
		Name:   "noop.price-gb",
		Usage:  "Sets the price of the noop service per GiB.",
		Hidden: true,
	}
	// FlagNoopAccessPolicies a comma-separated list of access policies that determines allowed identities to use the service.
	FlagNoopAccessPolicies = cli.StringFlag{
		Name:   "noop.access-policies",
		Usage:  "Comma separated list that determines the access policies of the noop service.",
		Hidden: true,
	}
)

// RegisterFlagsServiceNoop function register Wireguard flags to flag list
func RegisterFlagsServiceNoop(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagNoopPriceMinute,
		&FlagNoopPriceGB,
		&FlagNoopAccessPolicies,
	)
}

// ParseFlagsServiceNoop parses CLI flags and registers value to configuration
func ParseFlagsServiceNoop(ctx *cli.Context) {
	Current.ParseFloat64Flag(ctx, FlagNoopPriceMinute)
	Current.ParseFloat64Flag(ctx, FlagNoopPriceGB)
	Current.ParseStringFlag(ctx, FlagNoopAccessPolicies)
}
