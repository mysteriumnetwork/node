package communication

import (
	"encoding/json"
)

func NewCodecJSON() *codecJSON {
	return &codecJSON{}
}

type codecJSON struct{}

func (codec *codecJSON) Pack(payloadPtr interface{}) ([]byte, error) {
	return json.Marshal(payloadPtr)
}

func (codec *codecJSON) Unpack(data []byte, payloadPtr interface{}) error {
	return json.Unmarshal(data, payloadPtr)
}
