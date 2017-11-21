package identity

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/ethereum/go-ethereum/crypto"
)

func Test_CreateNewIdentity(t *testing.T) {
	id, err := CreateNewIdentity("testdata")
	assert.NoError(t, err)
	assert.Equal(t, len(id), 42)
}

func Test_GetIdentities(t *testing.T) {
	ids := GetIdentities("testdata")
	for _, id := range ids {
		fmt.Println(id)
	}
}

func Test_SignMessage(t *testing.T) {
	ids := GetIdentities("testdata")
	for _, id := range ids {
		signature, err := SignMessage("testdata", id, "message to sign")
		assert.NoError(t, err)
		assert.Equal(t, len(signature), 65)
	}
}

func Test_SignVerifyMessage(t *testing.T) {

	key, err := crypto.GenerateKey()
	assert.NoError(t, err)
	message := []byte("message to sign")

	signature, err := crypto.Sign(signHash(message), key)
	assert.NoError(t, err)

	rpk, err := crypto.Ecrecover(signHash(message), signature)
	assert.NoError(t, err)
	pubKey := crypto.ToECDSAPub(rpk)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	assert.Equal(t, recoveredAddr, crypto.PubkeyToAddress(key.PublicKey))

}