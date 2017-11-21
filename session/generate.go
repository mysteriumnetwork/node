package session

import (
	"crypto/rand"
	"encoding/hex"
)

type SessionId string

func GenerateSessionId() (sid SessionId, err error) {
	b, err := generateRandomBytes(16)

	return SessionId(hex.EncodeToString(b)), err
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}