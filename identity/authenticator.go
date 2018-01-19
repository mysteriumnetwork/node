package identity

import (
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
)

// Authenticator extracts identity
type Authenticator interface {
	Authenticate(message []byte, signature Signature) (Identity, error)
}

// NewAuthenticator extracts identity which was used to sign message
func NewAuthenticator() *authenticator {
	return &authenticator{}
}

type authenticator struct{}

func (authenticator *authenticator) Authenticate(message []byte, signature Signature) (Identity, error) {
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
