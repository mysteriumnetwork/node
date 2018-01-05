package identity

type VerifierFake struct{}

func (verifier *VerifierFake) Verify(message []byte, signature Signature) bool {
	signatureExpected := "signed" + string(message)

	return signature.String() == signatureExpected
}
