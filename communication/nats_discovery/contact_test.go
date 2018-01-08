package nats_discovery

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContactSerialize(t *testing.T) {
	var tests = []struct {
		model        ContactNATSV1
		expectedJson string
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
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
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
