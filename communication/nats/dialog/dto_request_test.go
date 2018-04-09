package dialog

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequestSerialize(t *testing.T) {
	var tests = []struct {
		model        dialogCreateRequest
		expectedJSON string
	}{
		{
			dialogCreateRequest{
				PeerID: "123",
			},
			`{
				"peer_id": "123"
			}`,
		},
		{
			dialogCreateRequest{},
			`{
				"peer_id": ""
			}`,
		},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.NoError(t, err)
		assert.JSONEq(t, test.expectedJSON, string(jsonBytes))
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
				"peer_id": "123"
			}`,
			dialogCreateRequest{
				PeerID: "123",
			},
			nil,
		},
		{
			`{}`,
			dialogCreateRequest{
				PeerID: "",
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
