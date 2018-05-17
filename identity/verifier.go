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

package identity

// Verifier checks message's sanity
type Verifier interface {
	Verify(message []byte, signature Signature) bool
}

// NewVerifierSigned constructs Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
func NewVerifierSigned() *verifierSigned {
	return &verifierSigned{NewExtractor()}
}

// NewVerifierIdentity constructs Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
//   - checks if message is from exact identity
func NewVerifierIdentity(peerID Identity) *verifierIdentity {
	return &verifierIdentity{NewExtractor(), peerID}
}

type verifierSigned struct {
	extractor Extractor
}

func (verifier *verifierSigned) Verify(message []byte, signature Signature) bool {
	_, err := verifier.extractor.Extract(message, signature)
	return err == nil
}

type verifierIdentity struct {
	extractor Extractor
	peerID    Identity
}

func (verifier *verifierIdentity) Verify(message []byte, signature Signature) bool {
	identity, err := verifier.extractor.Extract(message, signature)
	if err != nil {
		return false
	}

	return identity == verifier.peerID
}
