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

package tls

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type TLSPresharedKey []byte

func (key TLSPresharedKey) ToPEMFormat() string {
	buffer := bytes.Buffer{}

	fmt.Fprintln(&buffer, "-----BEGIN OpenVPN Static key V1-----")
	fmt.Fprintln(&buffer, hex.EncodeToString(key))
	fmt.Fprintln(&buffer, "-----END OpenVPN Static key V1-----")

	return buffer.String()
}

// createTLSCryptKey generates symmetric key in HEX format 2048 bits length
func createTLSCryptKey() (TLSPresharedKey, error) {

	taKey := make([]byte, 256)
	_, err := rand.Read(taKey)
	if err != nil {
		return nil, err
	}
	return TLSPresharedKey(taKey), nil
}
