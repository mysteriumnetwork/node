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

	"github.com/mysterium/node/datasize"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var (
	locationUS = dto_discovery.Location{
		Country: "US",
	}
	protocol = "tcp"
)

func TestServiceDefinitionSerialize(t *testing.T) {
	var tests = []struct {
		model        ServiceDefinition
		expectedJSON string
	}{
		{
			ServiceDefinition{
				Location:          locationUS,
				LocationOriginate: locationUS,
				SessionBandwidth:  Bandwidth(10 * datasize.Bit),
				Protocol:          protocol,
			},
			`{
				"location": {
					"country": "US"
				},
				"location_originate": {
					"country": "US"
				},
				"session_bandwidth": 10,
				"protocol": "tcp"
			}`,
		},
		{
			ServiceDefinition{},
			`{
				"location": {},
				"location_originate": {}
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestServiceDefinitionUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel ServiceDefinition
		expectedError error
	}{
		{
			`{
				"location": {
					"country": "US"
				},
				"location_originate": {
					"country": "US"
				},
				"session_bandwidth": 8,
				"protocol": "tcp"
			}`,
			ServiceDefinition{
				Location:          locationUS,
				LocationOriginate: locationUS,
				SessionBandwidth:  Bandwidth(1 * datasize.Byte),
				Protocol:          protocol,
			},
			nil,
		},
		{
			`{
				"location": {},
				"location_originate": {}
			}`,
			ServiceDefinition{},
			nil,
		},
		{
			`{}`,
			ServiceDefinition{},
			nil,
		},
	}

	for _, test := range tests {
		var model ServiceDefinition
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
