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
			dialogCreateResponse{
				Accepted: true,
			},
			`{
				"accepted": true
			}`,
		},
		{
			dialogCreateResponse{},
			`{
				"accepted": false
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
				"accepted": true
			}`,
			dialogCreateResponse{
				Accepted: true,
			},
			nil,
		},
		{
			`{
				"accepted": false
			}`,
			dialogCreateResponse{
				Accepted: false,
			},
			nil,
		},
		{
			`{
				"accepted": "true"
			}`,
			dialogCreateResponse{
				Accepted: false,
			},
			errors.New("json: cannot unmarshal string into Go struct field dialogCreateResponse.accepted of type bool"),
		},
		{
			`{}`,
			dialogCreateResponse{
				Accepted: false,
			},
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
