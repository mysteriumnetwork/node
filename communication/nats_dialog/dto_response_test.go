package nats_dialog

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResponseSerialize(t *testing.T) {
	var tests = []struct {
		model        dialogCreateResponse
		expectedJson string
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
				"reasonMessage": "Invalid identity"
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.NoError(t, err)
		assert.JSONEq(t, test.expectedJson, string(jsonBytes))
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
