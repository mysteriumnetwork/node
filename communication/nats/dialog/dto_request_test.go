package dialog

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequestSerialize(t *testing.T) {
	var tests = []struct {
		model        dialogCreateRequest
		expectedJson string
	}{
		{
			dialogCreateRequest{
				IdentityID: "123",
			},
			`{
				"identity_id": "123"
			}`,
		},
		{
			dialogCreateRequest{},
			`{
				"identity_id": ""
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.NoError(t, err)
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
	}
}

func TestRequestUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel dialogCreateRequest
		expectedError error
	}{
		{
			`{
				"identity_id": "123"
			}`,
			dialogCreateRequest{
				IdentityID: "123",
			},
			nil,
		},
		{
			`{}`,
			dialogCreateRequest{
				IdentityID: "",
			},
			nil,
		},
	}

	for _, test := range tests {
		var model dialogCreateRequest
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Exactly(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
