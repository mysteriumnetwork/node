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

package services

import (
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/urfave/cli/v2"
)

// GetStartOptions returns options to use for starting a service.
func GetStartOptions(serviceType string) (opts StartOptions, err error) {
	opts.TypeOptions, err = TypeConfiguredOptions(serviceType)
	if err != nil {
		return StartOptions{
			AccessPolicyList: nil,
			TypeOptions:      nil,
		}, err
	}

	switch serviceType {
	case openvpn.ServiceType:
		opts.AccessPolicyList = getPolicies(config.FlagOpenVPNAccessPolicies, config.FlagAccessPolicyList)
	case wireguard.ServiceType:
		opts.AccessPolicyList = getPolicies(config.FlagWireguardAccessPolicies, config.FlagAccessPolicyList)
	case noop.ServiceType:
		opts.AccessPolicyList = getPolicies(config.FlagNoopAccessPolicies, config.FlagAccessPolicyList)
	case scraping.ServiceType:
		opts.AccessPolicyList = []string{"mysterium"}
	case datatransfer.ServiceType:
		opts.AccessPolicyList = []string{"mysterium"}
	case dvpn.ServiceType:
		opts.AccessPolicyList = []string{"mysterium"}
	}
	return opts, nil
}

func getPolicies(flag cli.StringFlag, fallback cli.StringFlag) []string {
	policiesStr := config.GetString(flag)
	if policiesStr == "" {
		policiesStr = config.GetString(fallback)
	}

	var policies []string
	if len(policiesStr) > 0 {
		policies = strings.Split(policiesStr, ",")
	} else {
		policies = []string{}
	}
	return policies
}

// StartOptions describes options shared among multiple services
type StartOptions struct {
	AccessPolicyList []string
	TypeOptions      service.Options
}
