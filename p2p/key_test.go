/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	for i := 1; i < 20; i++ {
		pub1, priv1, err := GenerateKey()
		assert.NoError(t, err)
		pub2, priv2, err := GenerateKey()
		assert.NoError(t, err)
		assert.NotEqual(t, pub1, pub2)
		assert.NotEqual(t, priv1, priv2)
	}
}

func TestDecodePublicKey(t *testing.T) {
	expectedKey := PublicKey{0x4, 0x95, 0xb, 0x1a, 0xba, 0x3f, 0xff, 0xaa, 0xff, 0x5b, 0x81, 0x76, 0xe2, 0x55, 0xb4, 0x37, 0xc3, 0xba, 0xcf, 0x8e, 0xad, 0xc4, 0x70, 0x60, 0xe, 0xa5, 0xfd, 0xe6, 0x25, 0x2a, 0x23, 0x33}
	publicKey, err := DecodePublicKey("04950b1aba3fffaaff5b8176e255b437c3bacf8eadc470600ea5fde6252a2333")
	assert.NoError(t, err)
	assert.Equal(t, expectedKey, publicKey)
}

func TestEncryptDecrypt(t *testing.T) {
	pub1, priv1, _ := GenerateKey()
	pub2, priv2, _ := GenerateKey()

	msg := "Please hide me in ciphertext"
	ciphertext, err := priv1.Encrypt(pub2, []byte(msg))
	assert.NoError(t, err)

	plaintext, err := priv2.Decrypt(pub1, ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, msg, string(plaintext))
}
