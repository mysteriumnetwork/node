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

package communication

import (
	"encoding/json"
)

// NewCodecJSON returns codec which:
//   - encodes/decodes payloads forward & backward JSON format
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
