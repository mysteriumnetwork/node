package nats_dialog

import (
	"encoding/json"
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

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
		Signature: signature.Base64Encode(),
	})
}

func (codec *codecSecured) Unpack(data []byte, payloadPtr interface{}) error {
	envelope := &messageEnvelope{}
	err := codec.codecPacker.Unpack(data, envelope)
	if err != nil {
		return err
	}

	if !codec.verifier.Verify(envelope.Payload, identity.SignatureBase64Decode(envelope.Signature)) {
		return fmt.Errorf("invalid message signature '%s'", envelope.Signature)
	}

	return codec.codecPacker.Unpack(envelope.Payload, payloadPtr)
}

type messageEnvelope struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}
