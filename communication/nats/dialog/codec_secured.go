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
	"encoding/json"
	"fmt"

	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

// NewCodecSecured returns codec which:
//   - encodes/decodes payloads with any packer codec
//   - wraps encoded message with signature
//   - verifiers decoded message's signature
func NewCodecSecured(
	codecPacker communication.Codec,
	signer identity.Signer,
	verifier identity.Verifier,
) *codecSecured {
	return &codecSecured{
		codecPacker: codecPacker,
		signer:      signer,
		verifier:    verifier,
	}
}

type codecSecured struct {
	codecPacker communication.Codec
	signer      identity.Signer
	verifier    identity.Verifier
}

func (codec *codecSecured) Pack(payloadPtr interface{}) ([]byte, error) {
	payloadData, err := codec.codecPacker.Pack(payloadPtr)
	if err != nil {
		return []byte{}, err
	}

	signature, err := codec.signer.Sign(payloadData)
	if err != nil {
		return []byte{}, err
	}

	return codec.codecPacker.Pack(&messageEnvelope{
		Payload:   payloadData,
		Signature: signature.Base64(),
	})
}

func (codec *codecSecured) Unpack(data []byte, payloadPtr interface{}) error {
	envelope := &messageEnvelope{}
	err := codec.codecPacker.Unpack(data, envelope)
	if err != nil {
		return err
	}

	if !codec.verifier.Verify(envelope.Payload, identity.SignatureBase64(envelope.Signature)) {
		return fmt.Errorf("invalid message signature '%s'", envelope.Signature)
	}

	return codec.codecPacker.Unpack(envelope.Payload, payloadPtr)
}

type messageEnvelope struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}
