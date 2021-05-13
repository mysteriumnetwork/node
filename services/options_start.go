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
	"math/big"
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/urfave/cli/v2"
)

// GetStartOptions returns options to use for starting a service.
func GetStartOptions(serviceType string) (opts StartOptions, err error) {
	opts.TypeOptions, err = TypeConfiguredOptions(serviceType)
	if err != nil {
		return
	}

	switch serviceType {
	case openvpn.ServiceType:
		opts.PaymentPriceGiB = getPrice(config.FlagOpenVPNPriceGiB, config.FlagPaymentPriceGiB)
		opts.PaymentPriceHour = getPrice(config.FlagOpenVPNPriceHour, config.FlagPaymentPriceHour)
		opts.AccessPolicyList = getPolicies(config.FlagOpenVPNAccessPolicies, config.FlagAccessPolicyList)
	case wireguard.ServiceType:
		opts.PaymentPriceGiB = getPrice(config.FlagWireguardPriceGiB, config.FlagPaymentPriceGiB)
		opts.PaymentPriceHour = getPrice(config.FlagWireguardPriceHour, config.FlagPaymentPriceHour)
		opts.AccessPolicyList = getPolicies(config.FlagWireguardAccessPolicies, config.FlagAccessPolicyList)
	case noop.ServiceType:
		opts.PaymentPriceGiB = getPrice(config.FlagNoopPriceGB, config.FlagPaymentPriceGiB)
		opts.PaymentPriceHour = getPrice(config.FlagNoopPriceHour, config.FlagPaymentPriceHour)
		opts.AccessPolicyList = getPolicies(config.FlagNoopAccessPolicies, config.FlagAccessPolicyList)
	}
	return opts, nil
}

func getPrice(flag cli.Float64Flag, fallback cli.Float64Flag) *big.Int {
	value := config.GetFloat64(flag)
	if value == 0 {
		value = config.GetFloat64(fallback)
	}
	res, _ := new(big.Float).Mul(big.NewFloat(value), new(big.Float).SetInt(money.MystSize)).Int(nil)
	return res
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
	PaymentPriceGiB  *big.Int
	PaymentPriceHour *big.Int
	AccessPolicyList []string
	TypeOptions      service.Options
}
