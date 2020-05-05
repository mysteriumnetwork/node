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
	"github.com/mysteriumnetwork/node/money"
)

// GetStartOptions returns options to use for starting a service.
func GetStartOptions(serviceType string) (StartOptions, error) {
	typeOptions, err := TypeConfiguredOptions(serviceType)
	if err != nil {
		return StartOptions{}, err
	}

	policiesStr := config.GetString(config.FlagAccessPolicyList)
	var policies []string
	if len(policiesStr) > 0 {
		policies = strings.Split(policiesStr, ",")
	} else {
		policies = []string{}
	}

	return StartOptions{
		PaymentPricePerGB:     uint64(config.GetFloat64(config.FlagPaymentPricePerGB) * money.MystSize),
		PaymentPricePerMinute: uint64(config.GetFloat64(config.FlagPaymentPricePerMinute) * money.MystSize),
		AccessPolicyList:      policies,
		TypeOptions:           typeOptions,
	}, nil
}

// StartOptions describes options shared among multiple services
type StartOptions struct {
	PaymentPricePerGB     uint64
	PaymentPricePerMinute uint64
	AccessPolicyList      []string
	TypeOptions           service.Options
}
