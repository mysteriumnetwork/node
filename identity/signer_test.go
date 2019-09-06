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

package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	ks := NewKeystoreFilesystem("test_data", true)

	manager := NewIdentityManager(ks, nil)
	err := manager.Unlock("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68", "")
	assert.NoError(t, err)

	signer := NewSigner(ks, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	message := []byte("MystVpnSessionId:Boop!")
	signature, err := signer.Sign([]byte(message))
	signatureBase64 := signature.Base64()
	t.Logf("signature in base64: %s", signatureBase64)
	assert.NoError(t, err)
	assert.Equal(
		t,
		SignatureBase64("V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE="),
		signature,
	)
}
