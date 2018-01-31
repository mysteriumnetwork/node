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
	mockManager := &mockSessionManager{}
	mockExtractor := &mockExtractor{}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	mockManager := mockSessionManager{
		session.Session{
			ID:         session.SessionID("fake-id"),
			Config:     "vpn-session-configuration-string",
			ConsumerID: identity.FromAddress("deadbeef"),
		},
		true,
	}
	mockExtractor := &mockExtractor{
		identity.FromAddress("wrongsignature"),
		nil,
	}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestAuthenticatorReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	mockManager := mockSessionManager{
		session.Session{
			ID:         session.SessionID("fake-id"),
			Config:     "vpn-session-configuration-string",
			ConsumerID: identity.FromAddress("deadbeef"),
		},
		true,
	}
	mockExtractor := &mockExtractor{
		identity.FromAddress("deadbeef"),
		nil,
	}
	validateCredentials := NewSessionValidator(mockManager.FindSession, mockExtractor)

	authenticated, err := validateCredentials("not important", "not important")
	assert.NoError(t, err)
	assert.True(t, authenticated)

}

type mockSessionManager struct {
	onFindReturnSession session.Session
	onFindReturnSuccess bool
}

func (manager *mockSessionManager) FindSession(sessionId session.SessionID) (session.Session, bool) {
	return manager.onFindReturnSession, manager.onFindReturnSuccess
}

type mockExtractor struct {
	onExtractReturnIdentity identity.Identity
	onExtractReturnError    error
}

func (extractor *mockExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.onExtractReturnIdentity, extractor.onExtractReturnError
}
