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

package connection

import (
	"math/big"
	"testing"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/stretchr/testify/assert"
)

func Test_pricePerMinute(t *testing.T) {
	for _, test := range []struct {
		give   contract.PaymentMethodDTO
		expect money.Money
	}{
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerSeconds: 3000,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(500000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(600000000000000),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerSeconds: 6000,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(500000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(300000000000000),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerSeconds: 3000,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(1000000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(1200000000000000),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerSeconds: 0,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(1000000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(0),
				Currency: money.CurrencyMyst,
			},
		},
	} {
		got := aproxPricePerHour(test.give)
		assert.Equal(t, test.expect, got)
	}
}

func Test_pricePerGB(t *testing.T) {
	for _, test := range []struct {
		give   contract.PaymentMethodDTO
		expect money.Money
	}{
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerBytes: 5368709,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(500000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(100000002235174229),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerBytes: 2684354,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(500000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(200000041723260046),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerBytes: 5368709,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(1000000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(200000004470348458),
				Currency: money.CurrencyMyst,
			},
		},
		{
			give: contract.PaymentMethodDTO{
				Rate: contract.PaymentRateDTO{
					PerBytes: 0,
				},
				Price: money.Money{
					Amount:   new(big.Int).SetInt64(1000000000000000),
					Currency: money.CurrencyMyst,
				},
			},
			expect: money.Money{
				Amount:   new(big.Int).SetInt64(0),
				Currency: money.CurrencyMyst,
			},
		},
	} {
		got := aproxPricePerGB(test.give)
		assert.Equal(t, test.expect, got)
	}
}
