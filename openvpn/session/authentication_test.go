package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var returnSessionNotFound = func(sessionId session.SessionID) (session.Session, bool) {
	return session.Session{}, false
}

var returnSessionFound = func(sessionId session.SessionID) (session.Session, bool) {
	return session.Session{
			ID:         session.SessionID("fake-id"),
			Config:     "vpn-session-configuration-string",
			ConsumerID: identity.FromAddress("deadbeef"),
		},
		true
}

func TestAuthenticatorReturnsFalseWhenNoSessionFound(t *testing.T) {
	validateCredentials := NewSessionValidator(returnSessionNotFound, &mockExtractor{})

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	validateCredentials := NewSessionValidator(returnSessionFound, &mockExtractor{identity.FromAddress("wrongsignature"), nil})

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	validateCredentials := NewSessionValidator(returnSessionFound, &mockExtractor{identity.FromAddress("deadbeef"), nil})

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.True(t, authenticated)

}

type mockExtractor struct {
	onExtractReturnIdentity identity.Identity
	onExtractReturnError    error
}

func (mv *mockExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return mv.onExtractReturnIdentity, mv.onExtractReturnError
}
