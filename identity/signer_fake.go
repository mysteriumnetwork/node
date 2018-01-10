package identity

type SignerFake struct {
	ErrorMock error
}

func (signer *SignerFake) Sign(message []byte) (Signature, error) {
	if signer.ErrorMock != nil {
		return Signature{}, signer.ErrorMock
	}

	signatureBytes := messageFakeHash(message)
	return SignatureBytes(signatureBytes), nil
}

func messageFakeHash(message []byte) []byte {
	signatureBytes := []byte("signed")
	signatureBytes = append(signatureBytes, message...)

	return signatureBytes
}
