/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Info contains codeVerifier and codeChallenge
type Info struct {
	CodeVerifier  string
	CodeChallenge string
}

// Base64URLCodeVerifier encode to base64url for transport
func (info Info) Base64URLCodeVerifier() string {
	return base64.RawURLEncoding.EncodeToString([]byte(info.CodeVerifier))
}

// New returns a new set of codeVerifier and codeChallenge
// https://www.rfc-editor.org/rfc/rfc7636
func New(l uint) (Info, error) {
	verifier, err := CodeVerifier(l)
	if err != nil {
		return Info{}, err
	}

	challenge := ChallengeSHA256(verifier)

	return Info{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}, nil
}

// CodeVerifier creates a codeVerifier
// https://www.rfc-editor.org/rfc/rfc7636#section-4.1
// using rand.Reader code verifier is generated from subset of [A-Z] / [a-z] / [0-9]
func CodeVerifier(l uint) (string, error) {
	if l < 43 || l > 128 {
		return "", errors.New("l must be between [43;128]")
	}

	buf := make([]byte, l)
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		return "", fmt.Errorf("could not generate PKCE code: %w", err)
	}

	return string(buf[:]), nil
}

// ChallengeSHA256 generate codeChallenge from codeVerifier using sha256
// also encodes it base64
// https://www.rfc-editor.org/rfc/rfc7636#section-4.2
func ChallengeSHA256(verifier string) string {
	shaBytes := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(shaBytes[:])
}
