package identity

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyFunctionReturnsTrueWhenSignatureIsCorrect(t *testing.T) {
	message := []byte("I am message!")

	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	hash := messageHash(message)
	signature, err := crypto.Sign(hash, privateKey)
	assert.NoError(t, err)

	identity := toIdentity(privateKey)
	assert.True(t, NewVerifier(identity).Verify(message, signature))
}

func TestVerifyFunctionReturnsFalseWhenSignatureIsIncorrect(t *testing.T) {
	message := []byte("I am message!")
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	signature, err := crypto.Sign(messageHash(message), privateKey)
	assert.NoError(t, err)
	//change message
	message[1] = 'b'

	assert.False(t, NewVerifier(toIdentity(privateKey)).Verify(message, signature))
}

func toIdentity(privKey *ecdsa.PrivateKey) Identity {

	ecdsaPubKey := privKey.PublicKey

	return FromAddress(crypto.PubkeyToAddress(ecdsaPubKey).Hex())
}
