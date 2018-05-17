/*
 * Copyright (C) 2018 The Mysterium Network Authors
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

package discovery

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContactSerialize(t *testing.T) {
	var tests = []struct {
		model        ContactNATSV1
		expectedJSON string
	}{
		{
			ContactNATSV1{
				Topic:           "topic1234",
				BrokerAddresses: []string{"nats://far-server:4222"},
			},
			`{
				"topic": "topic1234",
				"broker_addresses":["nats://far-server:4222"]
			}`,
		},
		{
			ContactNATSV1{
				Topic:           "topic1234",
				BrokerAddresses: []string{"nats://far-server1:4222", "nats://far-server2:4222"},
			},
			`{
				"topic": "topic1234",
				"broker_addresses":["nats://far-server1:4222", "nats://far-server2:4222"]
			}`,
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
		expectedModel ContactNATSV1
		expectedError error
	}{
		{
			`{
				"topic": "topic1234",
				"broker_addresses":["nats://far-server1:4222"]
			}`,
			ContactNATSV1{
				Topic:           "topic1234",
				BrokerAddresses: []string{"nats://far-server1:4222"},
			},
			nil,
		},
		{
			`{
				"topic": "topic1234",
				"broker_addresses":["nats://far-server1:4222", "nats://far-server2:4222"]
			}`,
			ContactNATSV1{
				Topic:           "topic1234",
				BrokerAddresses: []string{"nats://far-server1:4222", "nats://far-server2:4222"},
			},
			nil,
		},
	}

	for _, test := range tests {
		var model ContactNATSV1
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
