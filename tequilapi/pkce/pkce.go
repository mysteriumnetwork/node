package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"io"
)

// Info contains codeVerifier and codeChallenge
type Info struct {
	codeVerifier  string
	codeChallenge string
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
		codeVerifier:  verifier,
		codeChallenge: challenge,
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
