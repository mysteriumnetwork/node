package dto

import (
	"encoding/json"
	"github.com/mysterium/node/datasize"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
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
		expectedJson string
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
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
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
				Protocol: 		   protocol,
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
