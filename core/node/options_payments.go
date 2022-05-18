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

package node

import (
	"math/big"
	"time"
)

// OptionsPayments controls the behaviour of payments
type OptionsPayments struct {
	MaxAllowedPaymentPercentile    int
	BCTimeout                      time.Duration
	HermesPromiseSettlingThreshold float64
	MaxFeeSettlingThreshold        float64
	SettlementTimeout              time.Duration
	SettlementRecheckInterval      time.Duration
	ConsumerDataLeewayMegabytes    uint64
	HermesStatusRecheckInterval    time.Duration
	BalanceFastPollInterval        time.Duration
	BalanceFastPollTimeout         time.Duration
	BalanceLongPollInterval        time.Duration
	RegistryTransactorPollInterval time.Duration
	RegistryTransactorPollTimeout  time.Duration
	MinAutoSettleAmount            float64
	MaxUnSettledAmount             float64

	ProviderInvoiceFrequency      time.Duration
	ProviderLimitInvoiceFrequency time.Duration

	MaxUnpaidInvoiceValue   *big.Int
	LimitUnpaidInvoiceValue *big.Int
}
