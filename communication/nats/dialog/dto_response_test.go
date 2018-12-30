/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dialog

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseSerialize(t *testing.T) {
	var tests = []struct {
		model        dialogCreateResponse
		expectedJSON string
	}{
		{
			responseOK,
			`{
				"reason": 200,
				"reasonMessage": "OK"
			}`,
		},
		{
			responseInvalidIdentity,
			`{
				"reason": 400,
				"reasonMessage": "Invalid Identity"
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.NoError(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestResponseUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel dialogCreateResponse
		expectedError error
	}{
		{
			`{
				"reason": 200,
				"reasonMessage": "OK"
			}`,
			dialogCreateResponse{
				Reason:        200,
				ReasonMessage: "OK",
			},
			nil,
		},
		{
			`{
				"reason": 500,
				"reasonMessage": "Bla"
			}`,
			dialogCreateResponse{
				Reason:        500,
				ReasonMessage: "Bla",
			},
			nil,
		},
		{
			`{
				"reason": true
			}`,
			dialogCreateResponse{},
			errors.New("json: cannot unmarshal bool into Go struct field dialogCreateResponse.reason of type uint"),
		},
		{
			`{}`,
			dialogCreateResponse{},
			nil,
		},
	}

	for _, test := range tests {
		var model dialogCreateResponse
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Exactly(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
