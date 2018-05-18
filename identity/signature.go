/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package identity

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
)

// SignatureBytes constructs Signature structure instance from bytes
func SignatureBytes(signatureBytes []byte) Signature {
	return Signature{signatureBytes}
}

// SignatureHex returns Signature struct from hex string
func SignatureHex(signature string) Signature {
	signatureBytes, _ := hex.DecodeString(signature)
	return Signature{signatureBytes}
}

// SignatureBase64 decodes base64 string signature into Signature
func SignatureBase64(signature string) Signature {
	signatureBytes, _ := base64.StdEncoding.DecodeString(signature)
	return Signature{signatureBytes}
}

// Signature structure
type Signature struct {
	raw []byte
}

// Base64 encodes signature into Base64 format
func (signature *Signature) Base64() string {
	return base64.StdEncoding.EncodeToString(signature.Bytes())
}

// Bytes returns signature in raw bytes format
func (signature *Signature) Bytes() []byte {
	return signature.raw
}

// EqualsTo compares current signature with a given one
func (signature Signature) EqualsTo(other Signature) bool {
	return bytes.Equal(signature.raw, other.raw)
}
