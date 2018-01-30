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
