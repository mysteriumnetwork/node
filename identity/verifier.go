package identity

import (
	"github.com/ethereum/go-ethereum/crypto"
)

type Verifier interface {
	Verify(message []byte, signature Signature) bool
}

type ethereumVerifier struct {
	peerIdentity Identity
}

func NewVerifier(peerIdentity Identity) *ethereumVerifier {
	return &ethereumVerifier{peerIdentity}
}

func (ev *ethereumVerifier) Verify(message []byte, signature Signature) bool {
	signatureBytes := signature.Bytes()
	if len(signatureBytes) == 0 {
		return false
	}

	recoveredKey, err := crypto.Ecrecover(messageHash(message), signatureBytes)
	if err != nil {
		return false
	}
	recoveredAddress := crypto.PubkeyToAddress(*crypto.ToECDSAPub(recoveredKey)).Hex()

	return FromAddress(recoveredAddress) == ev.peerIdentity
}
