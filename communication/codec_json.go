package communication

import (
	"encoding/json"
)

func NewCodecJSON() *codecJSON {
	return &codecJSON{}
}

type codecJSON struct{}

func (packer *codecJSON) Pack(payloadPtr interface{}) ([]byte, error) {
	return json.Marshal(payloadPtr)
}

func (packer *codecJSON) Unpack(data []byte, payloadPtr interface{}) error {
	return json.Unmarshal(data, payloadPtr)
}
