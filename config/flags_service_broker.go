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
	// FlagBrokerPriceMinute sets the price per minute for provided broker service.
	FlagBrokerPriceMinute = cli.Float64Flag{
		Name:   "broker.price-minute",
		Usage:  "Sets the price of the broker service per minute.",
		Hidden: true,
	}
	// FlagBrokerPriceGB sets the price per GiB for provided OpenVPN service.
	FlagBrokerPriceGB = cli.Float64Flag{
		Name:   "broker.price-gb",
		Usage:  "Sets the price of the broker service per GiB.",
		Hidden: true,
	}
	// FlagBrokerAccessPolicies a comma-separated list of access policies that determines allowed identities to use the service.
	FlagBrokerAccessPolicies = cli.StringFlag{
		Name:   "broker.access-policies",
		Usage:  "Comma separated list that determines the access policies of the broker service.",
		Hidden: true,
	}
)

// RegisterFlagsServiceBroker function register Wireguard flags to flag list
func RegisterFlagsServiceBroker(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagBrokerPriceMinute,
		&FlagBrokerPriceGB,
		&FlagBrokerAccessPolicies,
	)
}

// ParseFlagsServiceBroker parses CLI flags and registers value to configuration
func ParseFlagsServiceBroker(ctx *cli.Context) {
	Current.ParseFloat64Flag(ctx, FlagBrokerPriceMinute)
	Current.ParseFloat64Flag(ctx, FlagBrokerPriceGB)
	Current.ParseStringFlag(ctx, FlagBrokerAccessPolicies)
}
