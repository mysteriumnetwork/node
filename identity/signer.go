package identity

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

type SignerFactory func(id Identity) Signer

type Signer interface {
	Sign(message []byte) (Signature, error)
}

type keystoreSigner struct {
	keystore keystoreInterface
	account  accounts.Account
}

func NewSigner(keystore keystoreInterface, identity Identity) Signer {
	account := identityToAccount(identity)

	return &keystoreSigner{
		keystore: keystore,
		account:  account,
	}
}

func (ksSigner *keystoreSigner) Sign(message []byte) (Signature, error) {
	signature, err := ksSigner.keystore.SignHash(ksSigner.account, messageHash(message))
	if err != nil {
		return Signature{}, err
	}

	return SignatureBytes(signature), nil
}

func messageHash(data []byte) []byte {
	return crypto.Keccak256(data)
}
