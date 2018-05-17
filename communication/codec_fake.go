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

// NewCodecFake returns codec which:
//   - allows to mock encoded/decoded payloads
func NewCodecFake() *codecFake {
	return &codecFake{}
}

type codecFake struct {
	PackLastPayload interface{}
	packMock        []byte

	UnpackLastData []byte
	unpackMock     interface{}
}

func (codec *codecFake) MockPackResult(data []byte) {
	codec.packMock = data
}

func (codec *codecFake) MockUnpackResult(payload interface{}) {
	codec.unpackMock = payload
}

func (codec *codecFake) Pack(payloadPtr interface{}) ([]byte, error) {
	codec.PackLastPayload = payloadPtr

	return codec.packMock, nil
}

func (codec *codecFake) Unpack(data []byte, payloadPtr interface{}) error {
	codec.UnpackLastData = data

	payloadPtr = codec.unpackMock
	return nil
}
