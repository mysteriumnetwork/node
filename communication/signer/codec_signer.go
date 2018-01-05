package communication

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

func NewCodecSigner(codecPacker communication.Codec, signer identity.Signer) *codecSigner {
	return &codecSigner{
		codecPacker: codecPacker,
		signer:      signer,
	}
}

type messageJSON struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

type codecSigner struct {
	codecPacker communication.Codec
	signer      identity.Signer
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

	return codec.codecPacker.Pack(&messageJSON{
		Payload:   payloadData,
		Signature: signature.Hex(),
	})
}

func (codec *codecSigner) Unpack(data []byte, payloadPtr interface{}) error {
	return codec.codecPacker.Unpack(data, payloadPtr)
}
