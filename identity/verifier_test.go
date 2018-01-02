package identity

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyFunctionReturnsTrueWhenSignatureIsCorrect(t *testing.T) {
	message := []byte("I am message!")

	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	hash := messageHash(message)
	signatureBytes, err := crypto.Sign(hash, privateKey)
	signature := hex.EncodeToString(signatureBytes)
	assert.NoError(t, err)

	identity := toIdentity(privateKey)
	assert.True(t, NewVerifier(identity).Verify(message, signature))
}

func TestVerifyFunctionReturnsFalseWhenSignatureIsIncorrect(t *testing.T) {
	message := []byte("I am message!")
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	signatureBytes, err := crypto.Sign(messageHash(message), privateKey)
	signature := hex.EncodeToString(signatureBytes)
	assert.NoError(t, err)
	//change message
	message[1] = 'b'

	assert.False(t, NewVerifier(toIdentity(privateKey)).Verify(message, signature))
}

func toIdentity(privKey *ecdsa.PrivateKey) Identity {

	ecdsaPubKey := privKey.PublicKey

	return FromAddress(crypto.PubkeyToAddress(ecdsaPubKey).Hex())
}
