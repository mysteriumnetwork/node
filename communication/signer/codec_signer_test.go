package communication

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"testing"
)

type customPayload struct {
	Field int
}

func TestCodecSigner_Interface(t *testing.T) {
	var _ communication.Codec = NewCodecSigner(communication.NewCodecJSON(), &identity.SignerFake{})
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
				"payload": {"Field":0},
				"signature": "7369676e65647b224669656c64223a3132337d"
			}`,
		},
	}

	codec := NewCodecSigner(
		communication.NewCodecJSON(),
		&identity.SignerFake{},
	)
	for _, tt := range table {
		data, err := codec.Pack(tt.payload)

		assert.NoError(t, err)
		assert.JSONEq(t, tt.expectedPayload, string(data))
	}
}

func TestCodecSigner_PackError(t *testing.T) {
	codec := NewCodecSigner(
		communication.NewCodecJSON(),
		&identity.SignerFake{ErrorMock: errors.New("Signing failed")},
	)

	data, err := codec.Pack("data")
	assert.EqualError(t, err, "Signing failed")
	assert.Equal(t, []byte{}, data)
}
