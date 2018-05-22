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
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	originalSignerID = FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68")
	hijackedSignerID = FromAddress("0xded9913d38bfe94845b9e21fd32f43d0240e2f34")
)

func TestAuthenticate_WhenSignatureIsCorrect(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, originalSignerID, signerID, "Original signer should be extracted")
}

func TestAuthenticate_WhenBase64MessageSignatureIsCorrect(t *testing.T) {
	message := []byte("MystVpnSessionId:Boop!")
	signature := SignatureBase64("V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, originalSignerID, signerID, "Extracted signer should match original signer")
}

func TestAuthenticate_WhenSignatureIsEmpty(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.EqualError(t, err, "empty signature")
	assert.Exactly(t, Identity{}, signerID)
}

func TestAuthenticate_WhenSignatureIsMalformed(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("7369676e6564")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.EqualError(t, err, "invalid signature length")
	assert.Exactly(t, Identity{}, signerID)
}

func TestAuthenticate_WhenMessageIsChanged(t *testing.T) {
	message := []byte("Boop changed!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.NotEqual(t, originalSignerID, signerID, "Original signer should not be extracted")
	assert.Exactly(t, hijackedSignerID, signerID, "Another signer extracted")
}
