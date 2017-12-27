package identity

import "github.com/ethereum/go-ethereum/crypto"

type Verifier interface {
	Verify(message []byte, signature []byte) bool
}

type ethereumVerfier struct {
}

func NewVerifier() *ethereumVerfier {
	return &ethereumVerfier{}
}

func (ev *ethereumVerfier) Verify(message []byte, signature []byte) bool {
	hash := messageHash(message)
	_, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		return false
	}
	return true
}
