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
