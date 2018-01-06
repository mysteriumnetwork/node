package nats_dialog

import (
	"encoding/json"
	"errors"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

func NewCodecSigner(
	codecPacker communication.Codec,
	signer identity.Signer,
	verifier identity.Verifier,
) *codecSigner {
	return &codecSigner{
		codecPacker: codecPacker,
		signer:      signer,
		verifier:    verifier,
	}
}

type codecSigner struct {
	codecPacker communication.Codec
	signer      identity.Signer
	verifier    identity.Verifier
}

func (codec *codecSigner) Pack(payloadPtr interface{}) ([]byte, error) {
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
		Signature: signature.Hex(),
	})
}

func (codec *codecSigner) Unpack(data []byte, payloadPtr interface{}) error {
	envelope := &messageEnvelope{}
	err := codec.codecPacker.Unpack(data, envelope)
	if err != nil {
		return err
	}

	if !codec.verifier.Verify(envelope.Payload, identity.SignatureHex(envelope.Signature)) {
		return errors.New("Invalid message signature: " + envelope.Signature)
	}

	return codec.codecPacker.Unpack(envelope.Payload, payloadPtr)
}

type messageEnvelope struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}
