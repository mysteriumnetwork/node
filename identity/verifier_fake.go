package identity

import "bytes"

type VerifierFake struct{}

func (verifier *VerifierFake) Verify(message []byte, signature Signature) bool {
	signatureExpected := messageFakeHash(message)
	return bytes.Equal(signature.raw, signatureExpected)
}
