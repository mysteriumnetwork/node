package dto

import (
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
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
		{
			`{
				"country": 1
			}`,
			Location{},
			errors.New("json: cannot unmarshal number into Go struct field Location.country of type string"),
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
