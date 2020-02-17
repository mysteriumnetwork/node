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

package noop

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/stretchr/testify/assert"
)

func Test_PaymentMethod_Serialize(t *testing.T) {
	price := money.NewMoney(50000000, money.CurrencyMyst)

	var tests = []struct {
		model        pingpong.PaymentMethod
		expectedJSON string
	}{
		{
			pingpong.PaymentMethod{
				Price: price,
			},
			`{
				"type":"", 
				"bytes":0, 
				"duration":0,
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				}
			}`,
		},
		{
			pingpong.PaymentMethod{},
			`{
				"type":"", 
				"bytes":0, 
				"duration":0,
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

func Test_PaymentMethod_Unserialize(t *testing.T) {
	price := money.NewMoney(50000000, money.CurrencyMyst)

	var tests = []struct {
		json          string
		expectedModel pingpong.PaymentMethod
		expectedError error
	}{
		{
			`{
				"type":"", 
				"bytes":0, 
				"duration":0,
				"price": {
					"amount": 50000000,
					"currency": "MYST"
				}
			}`,
			pingpong.PaymentMethod{
				Price: price,
			},
			nil,
		},
		{
			`{
				"type":"", 
				"bytes":0, 
				"duration":0,
				"price": {}
			}`,
			pingpong.PaymentMethod{},
			nil,
		},
		{

			`{
				"type":     "",
				"bytes":    0,
				"duration": 0
			}`,
			pingpong.PaymentMethod{},
			nil,
		},
	}

	for _, test := range tests {
		var model pingpong.PaymentMethod
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		assert.Equal(t, test.expectedError, err)
	}
}
