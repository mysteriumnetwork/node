package identity

type SignerFake struct {
	ErrorMock error
}

func (signer *SignerFake) Sign(message []byte) (Signature, error) {
	if signer.ErrorMock != nil {
		return Signature{}, signer.ErrorMock
	}

	signatureBytes := []byte("signed")
	signatureBytes = append(signatureBytes, message...)

	return SignatureBytes(signatureBytes), nil
}
