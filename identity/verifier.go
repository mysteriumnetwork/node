package identity

import (
	"github.com/ethereum/go-ethereum/crypto"
)

type Verifier interface {
	Verify(message []byte, signature []byte) bool
}

type ethereumVerifier struct {
	peerIdentity Identity
}

func NewVerifier(peerIdentity Identity) *ethereumVerifier {
	return &ethereumVerifier{peerIdentity}
}

func (ev *ethereumVerifier) Verify(message []byte, signature []byte) bool {
	hash := messageHash(message)
	pk, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		return false
	}
	recoveredAddress := crypto.PubkeyToAddress(*crypto.ToECDSAPub(pk)).Hex()
	return FromAddress(recoveredAddress) == ev.peerIdentity
}
