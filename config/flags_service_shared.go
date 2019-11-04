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
	"gopkg.in/urfave/cli.v1/altsrc"
)

// Options describes options shared among multiple services
type Options struct {
	AccessPolicies []string
	ShaperEnabled  bool
}

var (
	IdentityFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "identity",
		Usage: "Keystore's identity used to provide service. If not given identity will be created automatically",
		Value: "",
	})
	IdentityPassphraseFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "identity.passphrase",
		Usage: "Used to unlock keystore's identity",
		Value: "",
	})
	AgreedTermsConditionsFlag = altsrc.NewBoolFlag(cli.BoolFlag{
		Name:  "agreed-terms-and-conditions",
		Usage: "Agree with terms & conditions",
	})
	AccessPoliciesFlag = cli.StringFlag{
		Name:  "access-policy.list",
		Usage: "Comma separated list that determines the allowed identities on our service.",
		Value: "",
	}
	// ShaperEnabledFlag reflects configuration setting of shaper being enabled
	ShaperEnabledFlag = cli.BoolFlag{
		Name:  "shaper.enabled",
		Usage: "Limit service bandwidth",
	}
)

// RegisterFlagsServiceShared registers shared service CLI flags
func RegisterFlagsServiceShared(flags *[]cli.Flag) {
	*flags = append(*flags,
		AgreedTermsConditionsFlag,
		IdentityFlag,
		IdentityPassphraseFlag,
		AccessPoliciesFlag,
		ShaperEnabledFlag,
	)
}

// ParseFlagsServiceShared parses shared service CLI flags and registers values to the configuration
func ParseFlagsServiceShared(ctx *cli.Context) {
	Current.SetDefault(AccessPoliciesFlag.Name, "")
	Current.SetDefault(ShaperEnabledFlag.Name, false)
	SetStringFlag(Current, AccessPoliciesFlag.Name, ctx)
	SetBoolFlag(Current, ShaperEnabledFlag.Name, ctx)
}
