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
	"time"

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

// ServicesOptions describes options shared among multiple services
type ServicesOptions struct {
	AccessPolicyAddress       string
	AccessPolicyList          []string
	AccessPolicyFetchInterval time.Duration
	ShaperEnabled             bool
}

var (
	// FlagIdentity keystore's identity.
	FlagIdentity = cli.StringFlag{
		Name:  "identity",
		Usage: "Keystore's identity used to provide service. If not given identity will be created automatically",
		Value: "",
	}
	// FlagIdentityPassphrase passphrase to unlock the identity.
	FlagIdentityPassphrase = cli.StringFlag{
		Name:  "identity.passphrase",
		Usage: "Used to unlock keystore's identity",
		Value: "",
	}
	// FlagAgreedTermsConditions agree with terms & conditions.
	FlagAgreedTermsConditions = cli.BoolFlag{
		Name:  "agreed-terms-and-conditions",
		Usage: "Agree with terms & conditions",
	}
	// FlagAccessPolicyAddress Policy oracle URL for retrieving access policies.
	FlagAccessPolicyAddress = cli.StringFlag{
		Name:  "access-policy.address",
		Usage: "URL of policy oracle endpoint for retrieving lists of access policies",
		Value: metadata.DefaultNetwork.AccessPolicyOracleAddress,
	}
	// FlagAccessPolicyList a comma-separated list of access policies that determines allowed identities to use the service.
	FlagAccessPolicyList = cli.StringFlag{
		Name:  "access-policy.list",
		Usage: "Comma separated list that determines the access policies applied to provide service.",
		Value: "",
	}
	// FlagAccessPolicyFetchInterval policy list fetch interval.
	FlagAccessPolicyFetchInterval = cli.DurationFlag{
		Name:  "access-policy.fetch",
		Usage: `Proposal fetch interval { "30s", "3m", "1h20m30s" }`,
		Value: 10 * time.Minute,
	}
	// FlagShaperEnabled enables bandwidth limitation.
	FlagShaperEnabled = cli.BoolFlag{
		Name:  "shaper.enabled",
		Usage: "Limit service bandwidth",
	}
)

// RegisterFlagsServiceShared registers shared service CLI flags
func RegisterFlagsServiceShared(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagIdentity,
		&FlagIdentityPassphrase,
		&FlagAgreedTermsConditions,
		&FlagAccessPolicyAddress,
		&FlagAccessPolicyList,
		&FlagAccessPolicyFetchInterval,
		&FlagShaperEnabled,
	)
}

// ParseFlagsServiceShared parses shared service CLI flags and registers values to the configuration
func ParseFlagsServiceShared(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagIdentity)
	Current.ParseStringFlag(ctx, FlagIdentityPassphrase)
	Current.ParseBoolFlag(ctx, FlagAgreedTermsConditions)
	Current.ParseStringFlag(ctx, FlagAccessPolicyAddress)
	Current.ParseStringFlag(ctx, FlagAccessPolicyList)
	Current.ParseDurationFlag(ctx, FlagAccessPolicyFetchInterval)
	Current.ParseBoolFlag(ctx, FlagShaperEnabled)
}
