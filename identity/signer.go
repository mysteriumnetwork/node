package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
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

func messageHash(data []byte) []byte {
	return crypto.Keccak256(data)
}
