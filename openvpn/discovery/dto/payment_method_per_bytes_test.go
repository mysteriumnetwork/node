/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package dto

import (
	"encoding/json"
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/money"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	price = money.NewMoney(0.5, money.CURRENCY_MYST)
)

func TestPaymentMethodPerBytesSerialize(t *testing.T) {
	var tests = []struct {
		model        PaymentPerBytes
		expectedJSON string
	}{
		{
			PaymentPerBytes{
				Price: price,
				Bytes: datasize.Gigabyte,
			},
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"bytes": 8589934592
			}`,
		},
		{
			PaymentPerBytes{},
			`{
				"price": {}
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestPaymentMethodPerBytesUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel PaymentPerBytes
		expectedError error
	}{
		{
			`{
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				},
				"bytes": 8589934592
			}`,
			PaymentPerBytes{
				Price: price,
				Bytes: datasize.Gigabyte,
			},
			nil,
		},
		{
			`{
				"price": {},
				"bytes": 0
			}`,
			PaymentPerBytes{},
			nil,
		},
		{
			`{}`,
			PaymentPerBytes{},
			nil,
		},
	}

	for _, test := range tests {
		var model PaymentPerBytes
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
