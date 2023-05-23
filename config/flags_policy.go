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
	"time"

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

var (
	// FlagAccessPolicyAddress Trust oracle URL for retrieving access policies.
	FlagAccessPolicyAddress = cli.StringFlag{
		Name:  metadata.FlagNames.AccessPolicyOracleAddress,
		Usage: "URL of trust oracle endpoint for retrieving lists of access policies",
		Value: metadata.DefaultNetwork.AccessPolicyOracleAddress,
	}
	// FlagAccessPolicyFetchInterval policy list fetch interval.
	FlagAccessPolicyFetchInterval = cli.DurationFlag{
		Name:  "access-policy.fetch",
		Usage: `Proposal fetch interval { "30s", "3m", "1h20m30s" }`,
		Value: 10 * time.Minute,
	}
	// FlagAccessPolicyFetchingEnabled policy list fetch enable
	FlagAccessPolicyFetchingEnabled = cli.BoolFlag{
		Name:  "access-policy.fetching-enabled",
		Usage: "Enable periodic fetching of access policies and saving to memory (allows support for whitelist types other than identity)",
		Value: false,
	}
)

// RegisterFlagsPolicy function registers Policy Oracle flags to flag list.
func RegisterFlagsPolicy(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagAccessPolicyAddress,
		&FlagAccessPolicyFetchInterval,
		&FlagAccessPolicyFetchingEnabled,
	)
}

// ParseFlagsPolicy function fills in PolicyOracle options from CLI context.
func ParseFlagsPolicy(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagAccessPolicyAddress)
	Current.ParseDurationFlag(ctx, FlagAccessPolicyFetchInterval)
	Current.ParseBoolFlag(ctx, FlagAccessPolicyFetchingEnabled)
}
