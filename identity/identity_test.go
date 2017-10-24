package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_PublicKeyEncodeDecode(t *testing.T) {
	privateKey := GenerateKeys()
	encoded := EncodePublicKey(privateKey.PublicKey)
	decodedPublicKey := DecodePublicKey(encoded)
	assert.Equal(t, &privateKey.PublicKey, decodedPublicKey)
}

func Test_PrivateKeyEncodeDecode(t *testing.T) {
	privateKey := GenerateKeys()
	encoded := EncodePrivateKey(*privateKey)
	decodedPrivateKey := DecodePrivateKey(encoded)
	assert.Equal(t, privateKey, decodedPrivateKey)
}

func Test_SignVerify(t *testing.T) {
	privateKey := GenerateKeys()
	r, s, _ := Sign(privateKey, "message")
	verified := Verify(privateKey.PublicKey, "message", r, s)
	assert.True(t, verified)
}
