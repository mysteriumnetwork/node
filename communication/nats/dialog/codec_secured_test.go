/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dialog

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
				"signature": "c2lnbmVkImhlbGxvIFwibmFtZVwiIg=="
			}`,
		},
		{
			true,
			`{
				"payload": true,
				"signature": "c2lnbmVkdHJ1ZQ=="
			}`,
		},
		{
			nil,
			`{
				"payload": null,
				"signature": "c2lnbmVkbnVsbA=="
			}`,
		},
		{
			&customPayload{123},
			`{
				"payload": {"Field":123},
				"signature": "c2lnbmVkeyJGaWVsZCI6MTIzfQ=="
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
				"signature": "c2lnbmVkImhlbGxvIFwibmFtZVwiIg=="
			}`,
			typeString,
			`hello "name"`,
		},
		{
			`{
				"payload": true,
				"signature": "c2lnbmVkdHJ1ZQ=="
			}`,
			typeBool,
			true,
		},
		{
			`{
				"payload": null,
				"signature": "c2lnbmVkbnVsbA=="
			}`,
			typeBool,
			nil,
		},
		{
			`{
				"payload": {"Field":123},
				"signature": "c2lnbmVkeyJGaWVsZCI6MTIzfQ=="
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
			"invalid message signature ''",
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
			"invalid message signature '7369676e65642268656c6c6f205c226e616d655c2222'",
		},
		{
			`{
				"payload": "hello \"name\"",
				"signature": "7369676e65642268656c6c6f205c226e616d655c2222-change"
			}`,
			"invalid message signature '7369676e65642268656c6c6f205c226e616d655c2222-change'",
		},
		{
			`{
				"payload": true,
				"signature": "c2lnbmVkdHJ1ZQ=="
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
