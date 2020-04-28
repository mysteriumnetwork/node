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
	"time"

	"github.com/mysteriumnetwork/node/config"
)

// SharedConfiguredOptions returns effective shared service options
func SharedConfiguredOptions() SharedOptions {
	policiesStr := config.GetString(config.FlagAccessPolicyList)
	var policies []string
	if len(policiesStr) > 0 {
		policies = strings.Split(policiesStr, ",")
	} else {
		policies = []string{}
	}

	return SharedOptions{
		PaymentPricePerGB:         config.GetFloat64(config.FlagPaymentPricePerGB),
		PaymentPricePerMinute:     config.GetFloat64(config.FlagPaymentPricePerMinute),
		AccessPolicyAddress:       config.GetString(config.FlagAccessPolicyAddress),
		AccessPolicyList:          policies,
		AccessPolicyFetchInterval: config.GetDuration(config.FlagAccessPolicyFetchInterval),
		ShaperEnabled:             config.GetBool(config.FlagShaperEnabled),
	}
}

// SharedOptions describes options shared among multiple services
type SharedOptions struct {
	PaymentPricePerGB         float64
	PaymentPricePerMinute     float64
	AccessPolicyAddress       string
	AccessPolicyList          []string
	AccessPolicyFetchInterval time.Duration
	ShaperEnabled             bool
}
