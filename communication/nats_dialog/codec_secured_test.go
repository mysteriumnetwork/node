package nats_dialog

import (
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

type customPayload struct {
	Field int
}

func TestCodecSigner_Interface(t *testing.T) {
	var _ communication.Codec = NewCodecSecured(
		communication.NewCodecJSON(),
		&identity.SignerFake{},
		&identity.VerifierFake{},
	)
}

func TestCodecSigner_Pack(t *testing.T) {
	table := []struct {
		payload         interface{}
		expectedPayload string
	}{
		{
			`hello "name"`,
			`{
				"payload": "hello \"name\"",
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222"
			}`,
		},
		{
			true,
			`{
				"payload": true,
				"signature": "7369676e656474727565"
			}`,
		},
		{
			nil,
			`{
				"payload": null,
				"signature": "7369676e65646e756c6c"
			}`,
		},
		{
			&customPayload{123},
			`{
				"payload": {"Field":123},
				"signature": "7369676e65647b224669656c64223a3132337d"
			}`,
		},
	}

	codec := NewCodecSecured(
		communication.NewCodecJSON(),
		&identity.SignerFake{},
		&identity.VerifierFake{},
	)
	for _, tt := range table {
		data, err := codec.Pack(tt.payload)

		assert.NoError(t, err)
		assert.JSONEq(t, tt.expectedPayload, string(data))
	}
}

func TestCodecSigner_PackError(t *testing.T) {
	codec := NewCodecSecured(
		communication.NewCodecJSON(),
		&identity.SignerFake{ErrorMock: errors.New("Signing failed")},
		&identity.VerifierFake{},
	)

	data, err := codec.Pack("data")
	assert.EqualError(t, err, "Signing failed")
	assert.Equal(t, []byte{}, data)
}

const (
	typeString = iota
	typeBool
	typeCustom
)

func TestCodecSigner_Unpack(t *testing.T) {
	table := []struct {
		data            string
		payloadType     int
		expectedPayload interface{}
	}{
		{
			`{
				"payload": "hello \"name\"",
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222"
			}`,
			typeString,
			`hello "name"`,
		},
		{
			`{
				"payload": true,
				"signature": "7369676e656474727565"
			}`,
			typeBool,
			true,
		},
		{
			`{
				"payload": null,
				"signature": "7369676e65646e756c6c"
			}`,
			typeBool,
			nil,
		},
		{
			`{
				"payload": {"Field":123},
				"signature": "7369676e65647b224669656c64223a3132337d"
			}`,
			typeCustom,
			&customPayload{123},
		},
	}

	codec := NewCodecSecured(
		communication.NewCodecJSON(),
		&identity.SignerFake{},
		&identity.VerifierFake{},
	)
	for _, tt := range table {
		switch tt.payloadType {
		case typeString:
			var payload string
			err := codec.Unpack([]byte(tt.data), &payload)

			assert.NoError(t, err)
			assert.Exactly(t, tt.expectedPayload, payload)

		case typeBool:
			var payload interface{}
			err := codec.Unpack([]byte(tt.data), &payload)

			assert.NoError(t, err)
			assert.Exactly(t, tt.expectedPayload, payload)

		case typeCustom:
			var payload *customPayload
			err := codec.Unpack([]byte(tt.data), &payload)

			assert.NoError(t, err)
			assert.Exactly(t, tt.expectedPayload, payload)

		default:
			t.Fatalf("Unknown type: %d", tt.payloadType)
		}
	}
}

func TestCodecSigner_UnpackError(t *testing.T) {
	table := []struct {
		data          string
		expectedError string
	}{
		{
			`{
				"payload": "hello \"name\""
			}`,
			"Invalid message signature: ",
		},
		{
			`{
				"payload": hello name,
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222"
			}`,
			"invalid character 'h' looking for beginning of value",
		},
		{
			`{
				"payload": "hello \"name\" change",
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222"
			}`,
			"Invalid message signature: 7369676e65642268656c6c6f205c226e616d655c2222",
		},
		{
			`{
				"payload": "hello \"name\"",
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222-change"
			}`,
			"Invalid message signature: 7369676e65642268656c6c6f205c226e616d655c2222-change",
		},
		{
			`{
				"payload": true,
				"signature": "7369676e656474727565"
			}`,
			"json: cannot unmarshal bool into Go value of type string",
		},
	}

	codec := NewCodecSecured(
		communication.NewCodecJSON(),
		&identity.SignerFake{},
		&identity.VerifierFake{},
	)
	for _, tt := range table {
		var payload string
		err := codec.Unpack([]byte(tt.data), &payload)

		assert.EqualError(t, err, tt.expectedError)
		assert.Exactly(t, payload, "")
	}
}
