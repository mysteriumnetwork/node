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

package shared

import (
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/cliflags"
	"gopkg.in/urfave/cli.v1"
)

// Options describes options shared among multiple services
type Options struct {
	AccessPolicies []string
	ShaperEnabled  bool
}

var (
	accessPoliciesFlag = cli.StringFlag{
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

// RegisterFlags registers shared service CLI flags
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags,
		accessPoliciesFlag,
		ShaperEnabledFlag,
	)
}

// Configure parses shared service CLI flags and registers values to the configuration
func Configure(ctx *cli.Context) {
	configureDefaults()
	configureCLI(ctx)
}

func configureDefaults() {
	config.Current.SetDefault(accessPoliciesFlag.Name, "")
	config.Current.SetDefault(ShaperEnabledFlag.Name, false)
}

func configureCLI(ctx *cli.Context) {
	cliflags.SetString(config.Current, accessPoliciesFlag.Name, ctx)
	cliflags.SetBool(config.Current, ShaperEnabledFlag.Name, ctx)
}

// ConfiguredOptions returns effective shared service options
func ConfiguredOptions() Options {
	policiesStr := config.Current.GetString(accessPoliciesFlag.Name)
	var policies []string
	if len(policiesStr) > 0 {
		policies = strings.Split(policiesStr, ",")
	} else {
		policies = []string{}
	}
	return Options{
		AccessPolicies: policies,
		ShaperEnabled:  config.Current.GetBool(ShaperEnabledFlag.Name),
	}
}
