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

package promise

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/mysteriumnetwork/node/money"
	"github.com/stretchr/testify/assert"
)

func TestBalanceMessage_Serialize(t *testing.T) {
	var tests = []struct {
		model        BalanceMessage
		expectedJSON string
	}{
		{
			BalanceMessage{123, true, money.Money{}},
			`{
				"request_id": 123,
				"accepted": true,
				"balance": {}
			}`,
		},
		{
			BalanceMessage{0, false, money.Money{}},
			`{
				"request_id": 0,
				"accepted": false,
				"balance": {}
			}`,
		},
		{
			BalanceMessage{},
			`{
				"request_id": 0,
				"accepted": false,
				"balance": {}
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestBalanceMessage_Unserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel BalanceMessage
		expectedError error
	}{
		{
			`{
				"request_id": 123,
				"accepted": true,
				"balance": {}
			}`,
			BalanceMessage{123, true, money.Money{}},
			nil,
		},
		{
			`{}`,
			BalanceMessage{},
			nil,
		},
	}

	for _, test := range tests {
		var model BalanceMessage
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestBalanceMessage_UnserializeError(t *testing.T) {
	jsonData := []byte(`
		{"request_id": "6"
	}`)

	var model BalanceMessage
	err := json.Unmarshal(jsonData, &model)

	assert.IsType(t, &json.UnmarshalTypeError{}, err)
	assert.Regexp(t, regexp.MustCompile("^json: "), err.Error())
}
