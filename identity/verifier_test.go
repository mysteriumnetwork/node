package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifier_VerifyWhenSignatureIsCorrect(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifier(FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	assert.True(t, verifier.Verify(message, signature))
}

func TestVerifier_VerifyUppercasedIdentity(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifier(FromAddress("0x53A835143C0EF3BBCBFA796d7eb738CA7DD28F68"))
	assert.True(t, verifier.Verify(message, signature))
}

func TestVerifyFunctionReturnsFalseWhenSignatureIsIncorrect(t *testing.T) {
	message := []byte("boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	verifier := NewVerifier(FromAddress("0x53A835143C0EF3BBCBFA796d7eb738CA7DD28F68"))
	assert.False(t, verifier.Verify(message, signature))
}
