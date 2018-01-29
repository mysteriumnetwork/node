package identity

// Verifier checks message's sanity
type Verifier interface {
	Verify(message []byte, signature Signature) bool
}

// NewVerifierSigned construct Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
func NewVerifierSigned() *verifierSigned {
	return &verifierSigned{NewAuthenticator()}
}

// NewVerifierIdentity construct Verifier which:
//   - checks signature's sanity
//   - checks if message was unchanged by middleman
//   - checks if message is from exact identity
func NewVerifierIdentity(peerID Identity) *verifierIdentity {
	return &verifierIdentity{NewAuthenticator(), peerID}
}

type verifierSigned struct {
	authenticator Authenticator
}

func (verifier *verifierSigned) Verify(message []byte, signature Signature) bool {
	_, err := verifier.authenticator.Authenticate(message, signature)
	return err == nil
}

type verifierIdentity struct {
	authenticator Authenticator
	peerID        Identity
}

func (verifier *verifierIdentity) Verify(message []byte, signature Signature) bool {
	identity, err := verifier.authenticator.Authenticate(message, signature)
	if err != nil {
		return false
	}

	return identity == verifier.peerID
}
