package identity

type VerifierFake struct{}

func (verifier *VerifierFake) Verify(message []byte, signature Signature) bool {
	signatureExpected := messageFakeHash(message)
	return signature.EqualsTo(SignatureBytes(signatureExpected))
}
