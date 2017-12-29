package identity

import (
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
)

type Signer interface {
	Sign(message []byte) ([]byte, error)
}

type keystoreSigner struct {
	keystore keystoreInterface
	account  accounts.Account
}

func NewSigner(keystore keystoreInterface, identity Identity) Signer {
	return &keystoreSigner{
		keystore: keystore,
		account:  identityToAccount(identity),
	}
}

func (ksSigner *keystoreSigner) Sign(message []byte) ([]byte, error) {
	signature, err := ksSigner.keystore.SignHash(ksSigner.account, messageHash(message))
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// messageHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
// keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents keystoreSigner of transactions.
func messageHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)

	return crypto.Keccak256([]byte(msg))
}
