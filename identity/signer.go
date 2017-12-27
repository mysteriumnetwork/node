package identity

import (
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
)

type signerInterface interface {
	Sign(message []byte) ([]byte, error)
}

type signer struct {
	keystoreManager keystoreInterface
	identity        Identity
}

func newSigner(keystore keystoreInterface, identity Identity) signerInterface {
	return &signer{
		keystoreManager: keystore,
		identity:        identity,
	}
}

func (s *signer) Sign(message []byte) ([]byte, error) {
	keystoreManager := s.keystoreManager
	signature, err := keystoreManager.SignHash(identityToAccount(s.identity), signHash(message))
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signer of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)

	return crypto.Keccak256([]byte(msg))
}
