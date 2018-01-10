package identity

import (
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
)

type Verifier interface {
	Verify(message []byte, signature Signature) bool
}

func NewVerifierSigned() *verifierSigned {
	return &verifierSigned{}
}

func NewVerifierIdentity(peerIdentity Identity) *verifierIdentity {
	return &verifierIdentity{peerIdentity}
}

type verifierSigned struct{}

func (verifier *verifierSigned) Verify(message []byte, signature Signature) bool {
	_, err := extractSignerIdentity(message, signature)
	return err == nil
}

type verifierIdentity struct {
	peerIdentity Identity
}

func (verifier *verifierIdentity) Verify(message []byte, signature Signature) bool {
	identity, err := extractSignerIdentity(message, signature)
	if err != nil {
		return false
	}

	return identity == verifier.peerIdentity
}

func extractSignerIdentity(message []byte, signature Signature) (Identity, error) {
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
