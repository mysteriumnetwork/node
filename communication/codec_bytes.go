package communication

import (
	"fmt"
	"reflect"
)

// NewCodecBytes returns codec which:
//   - supports only byte payloads
//   - does not perform any fancy encoding/decoding on payloads
func NewCodecBytes() *codecBytes {
	return &codecBytes{}
}

type codecBytes struct{}

func (codec *codecBytes) Pack(payloadPtr interface{}) ([]byte, error) {
	if payloadPtr == nil {
		return []byte{}, nil
	}

	switch payload := payloadPtr.(type) {
	case []byte:
		return payload, nil

	case byte:
		return []byte{payload}, nil

	case string:
		return []byte(payload), nil
	}

	return []byte{}, fmt.Errorf("Cant pack payload: %#v", payloadPtr)
}

func (codec *codecBytes) Unpack(data []byte, payloadPtr interface{}) error {

	switch payload := payloadPtr.(type) {
	case *[]byte:
		*payload = data
		return nil

	default:
		payloadValue := reflect.ValueOf(payloadPtr)
		return fmt.Errorf("Cant unpack to payload: %s", payloadValue.Type().String())
	}
}
