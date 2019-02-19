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
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignerFactory callback returning Signer
type SignerFactory func(id Identity) Signer

// Signer interface signifies an ability to sign a message
type Signer interface {
	Sign(message []byte) (Signature, error)
}

type keystoreSigner struct {
	keystore Keystore
	account  accounts.Account
}

// NewSigner returns new instance of Signer
func NewSigner(keystore Keystore, identity Identity) Signer {
	account := identityToAccount(identity)

	return &keystoreSigner{
		keystore: keystore,
		account:  account,
	}
}

// Sign signs given message and returns signature
func (ksSigner *keystoreSigner) Sign(message []byte) (Signature, error) {
	signature, err := ksSigner.keystore.SignHash(ksSigner.account, messageHash(message))
	if err != nil {
		return Signature{}, err
	}

	return SignatureBytes(signature), nil
}

func messageHash(data []byte) []byte {
	return crypto.Keccak256(data)
}
