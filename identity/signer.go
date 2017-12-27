package identity

import (
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
)

type Signer interface {
	Sign(message []byte) ([]byte, error)
}

type keystoreSigner struct {
	keystoreManager keystoreInterface
	identity        Identity
}

func NewSigner(keystore keystoreInterface, identity Identity) Signer {
	return &keystoreSigner{
		keystoreManager: keystore,
		identity:        identity,
	}
}

func (ksSigner *keystoreSigner) Sign(message []byte) ([]byte, error) {
	signature, err := ksSigner.keystoreManager.SignHash(identityToAccount(ksSigner.identity), messageHash(message))
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// messageHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents keystoreSigner of transactions.
func messageHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)

	return crypto.Keccak256([]byte(msg))
}
