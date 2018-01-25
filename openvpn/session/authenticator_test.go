package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var sessionNotFoundFunction = func(sessionId session.SessionID) (session.Session, bool) {
	return session.Session{}, false
}

var sessionFoundFunction = func(sessionId session.SessionID) (session.Session, bool) {
	return session.Session{}, true
}

var validSignatureVerifierFactory = func(identity identity.Identity) identity.Verifier {
	return &mockVerifier{validSignature: true}
}

var invalidSignatureVerifierFactory = func(identity identity.Identity) identity.Verifier {
	return &mockVerifier{validSignature: false}
}

func TestAuthenticatorReturnsFalseWhenNoSessionFound(t *testing.T) {
	authenticator := NewSessionAuthenticator(sessionNotFoundFunction, validSignatureVerifierFactory)

	authenticated, err := authenticator.ValidateSession("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	authenticator := NewSessionAuthenticator(sessionFoundFunction, invalidSignatureVerifierFactory)

	authenticated, err := authenticator.ValidateSession("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	authenticator := NewSessionAuthenticator(sessionFoundFunction, validSignatureVerifierFactory)

	authenticated, err := authenticator.ValidateSession("not important", "not important")
	assert.NoError(t, err)
	assert.True(t, authenticated)

}

type mockVerifier struct {
	validSignature bool
}

func (mv *mockVerifier) Verify(message []byte, signature identity.Signature) bool {
	return mv.validSignature
}
