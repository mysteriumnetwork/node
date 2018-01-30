package identity

// Verifier checks message's sanity
type Verifier interface {
	Verify(message []byte, signature Signature) bool
}

// NewVerifierSigned constructs Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
func NewVerifierSigned() *verifierSigned {
	return &verifierSigned{NewExtractor()}
}

// NewVerifierIdentity constructs Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
//   - checks if message is from exact identity
func NewVerifierIdentity(peerID Identity) *verifierIdentity {
	return &verifierIdentity{NewExtractor(), peerID}
}

type verifierSigned struct {
	extractor Extractor
}

func (verifier *verifierSigned) Verify(message []byte, signature Signature) bool {
	_, err := verifier.extractor.Extract(message, signature)
	return err == nil
}

type verifierIdentity struct {
	extractor Extractor
	peerID    Identity
}

func (verifier *verifierIdentity) Verify(message []byte, signature Signature) bool {
	identity, err := verifier.extractor.Extract(message, signature)
	if err != nil {
		return false
	}

	return identity == verifier.peerID
}
