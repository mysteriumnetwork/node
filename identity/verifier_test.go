package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifier_extractSignerWhenSignatureIsCorrect(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	signerId, err := extractSignerIdentity(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"), signerId)
}

func TestVerifier_extractSignerWhenSignatureIsEmpty(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("")

	signerId, err := extractSignerIdentity(message, signature)
	assert.EqualError(t, err, "empty signature")
	assert.Exactly(t, Identity{}, signerId)
}

func TestVerifier_extractSignerWhenSignatureIsMalformed(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("7369676e6564")

	signerId, err := extractSignerIdentity(message, signature)
	assert.EqualError(t, err, "invalid signature length")
	assert.Exactly(t, Identity{}, signerId)
}

func TestVerifier_extractSignerWhenMessageIsChanged(t *testing.T) {
	message := []byte("Boop changed!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	signerId, err := extractSignerIdentity(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, FromAddress("0xded9913d38bfe94845b9e21fd32f43d0240e2f34"), signerId)
}

func TestVerifierIdentity_Verify(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifierIdentity(FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	assert.True(t, verifier.Verify(message, signature))
}

func TestVerifierIdentity_VerifyWithUppercaseIdentity(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifierIdentity(FromAddress("0x53A835143C0EF3BBCBFA796d7eb738CA7DD28F68"))
	assert.True(t, verifier.Verify(message, signature))
}

func TestVerifierIdentity_VerifyWhenWrongSender(t *testing.T) {
	message := []byte("boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifierIdentity(FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	assert.False(t, verifier.Verify(message, signature))
}

func TestVerifierSigned_Verify(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifierSigned()
	assert.True(t, verifier.Verify(message, signature))
}

func TestVerifierSigned_VerifyWhenMalformedSignature(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("7369676e6564")

	verifier := NewVerifierSigned()
	assert.False(t, verifier.Verify(message, signature))
}

func TestVerifierSigned_VerifyWhenMessageIsChanged(t *testing.T) {
	message := []byte("Boop changed!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifierSigned()
	assert.True(t, verifier.Verify(message, signature))
}
