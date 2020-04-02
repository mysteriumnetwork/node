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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/box"
)

const keySize = 32

// PublicKey represents p2p public key part.
type PublicKey [keySize]byte

// Hex returns public encoded as in hex string.
func (k *PublicKey) Hex() string {
	return hex.EncodeToString(k[:])
}

// DecodePublicKey converts hex string to PublicKey.
func DecodePublicKey(keyHex string) (PublicKey, error) {
	key := [keySize]byte{}
	src, err := hex.DecodeString(keyHex)
	if err != nil {
		return key, fmt.Errorf("could not decode key from hex: %w", err)
	}
	if len(src) != keySize {
		return key, fmt.Errorf("key size is invalid, expect %d, got %d", keySize, len(src))
	}
	copy(key[:], src)
	return key, nil
}

// PrivateKey represents p2p private key.
type PrivateKey [32]byte

// Encrypt encrypts data using nacl as underlying crypt system.
func (k *PrivateKey) Encrypt(publicKey PublicKey, src []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("could not read random nonce: %w", err)
	}
	return box.Seal(nonce[:], src, &nonce, (*[32]byte)(&publicKey), (*[32]byte)(k)), nil
}

// Decrypt decrypts data using nacl as underlying crypt system.
func (k *PrivateKey) Decrypt(publicKey PublicKey, ciphertext []byte) ([]byte, error) {
	var decryptNonce [24]byte
	copy(decryptNonce[:], ciphertext[:24])
	decrypted, ok := box.Open(nil, ciphertext[24:], &decryptNonce, (*[32]byte)(&publicKey), (*[32]byte)(k))
	if !ok {
		return nil, errors.New("could not decrypt message")
	}
	return decrypted, nil
}

// GenerateKey generates p2p public and private key pairs.
func GenerateKey() (PublicKey, PrivateKey, error) {
	publicKey := [keySize]byte{}
	privateKey := [keySize]byte{}
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return PublicKey{}, PrivateKey{}, fmt.Errorf("could not generate public private key pairs: %w", err)
	}
	copy(publicKey[:], pub[:])
	copy(privateKey[:], priv[:])
	return publicKey, privateKey, nil
}
