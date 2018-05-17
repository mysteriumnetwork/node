/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
