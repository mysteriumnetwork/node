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

package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationSerialize(t *testing.T) {
	var tests = []struct {
		model        Location
		expectedJSON string
	}{
		{
			Location{"XX", "YY", "AS123"},
			`{
				"country": "XX",
				"city": "YY",
				"asn": "AS123"
			}`,
		},
		{
			Location{Country: "XX"},
			`{
				"country": "XX"
			}`,
		},
		{
			Location{},
			`{}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestLocationUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel Location
		expectedError error
	}{
		{
			`{
				"country": "XX",
				"city": "YY",
				"asn": "AS123"
			}`,
			Location{"XX", "YY", "AS123"},
			nil,
		},
		{
			`{
				"country": "XX"
			}`,
			Location{Country: "XX"},
			nil,
		},
		{
			`{}`,
			Location{},
			nil,
		},
	}

	for _, test := range tests {
		var model Location
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
