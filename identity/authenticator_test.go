package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAuthenticate_WhenSignatureIsCorrect(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	authenticator := &authenticator{}
	signerId, err := authenticator.Authenticate(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"), signerId)
}

func TestAuthenticate_WhenSignatureIsEmpty(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("")

	authenticator := &authenticator{}
	signerId, err := authenticator.Authenticate(message, signature)
	assert.EqualError(t, err, "empty signature")
	assert.Exactly(t, Identity{}, signerId)
}

func TestAuthenticate_WhenSignatureIsMalformed(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("7369676e6564")

	authenticator := &authenticator{}
	signerId, err := authenticator.Authenticate(message, signature)
	assert.EqualError(t, err, "invalid signature length")
	assert.Exactly(t, Identity{}, signerId)
}

func TestAuthenticate_WhenMessageIsChanged(t *testing.T) {
	message := []byte("Boop changed!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	authenticator := &authenticator{}
	signerId, err := authenticator.Authenticate(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, FromAddress("0xded9913d38bfe94845b9e21fd32f43d0240e2f34"), signerId)
}
