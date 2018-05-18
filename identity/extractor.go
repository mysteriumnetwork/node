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
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
)

// Extractor is able to message signer's identity
type Extractor interface {
	Extract(message []byte, signature Signature) (Identity, error)
}

// NewExtractor constructs Extractor instance
func NewExtractor() *extractor {
	return &extractor{}
}

type extractor struct{}

// Extractor extracts identity which was used to sign given message
func (extractor *extractor) Extract(message []byte, signature Signature) (Identity, error) {
	signatureBytes := signature.Bytes()
	if len(signatureBytes) == 0 {
		return Identity{}, errors.New("empty signature")
	}

	recoveredKey, err := crypto.Ecrecover(messageHash(message), signatureBytes)
	if err != nil {
		return Identity{}, err
	}
	recoveredAddress := crypto.PubkeyToAddress(*crypto.ToECDSAPub(recoveredKey)).Hex()

	return FromAddress(recoveredAddress), nil
}
