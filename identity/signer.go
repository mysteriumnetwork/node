package identity

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	Sign(message []byte) (string, error)
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

func (ksSigner *keystoreSigner) Sign(message []byte) (string, error) {
	err := ksSigner.keystore.Unlock(ksSigner.account, "")
	if err != nil {
		return "", err
	}

	signature, err := ksSigner.keystore.SignHash(ksSigner.account, messageHash(message))
	if err != nil {
		return "", err
	}

	return signatureEncode(signature), nil
}

func messageHash(data []byte) []byte {
	return crypto.Keccak256(data)
}

func signatureEncode(signature []byte) string {
	return hex.EncodeToString(signature)
}

func signatureDecode(signature string) ([]byte, error) {
	return hex.DecodeString(signature)
}
