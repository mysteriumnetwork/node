package session

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/identity"
	"sync"
)

var mockManager = &MockSessionManager{
	session.Session{
		ID:         session.SessionID("fake-id"),
		Config:     mockedVPNConfig,
		ConsumerID: identity.FromAddress("deadbeef"),
	},
	true,
}

var mockExtractor = &MockIdentityExtractor{
	identity.FromAddress("deadbeef"),
	nil,
}

var fakeManager = NewManager(mockManager)

var mockValidator = NewValidator(fakeManager, mockExtractor)

func TestValidateReturnsFalseWhenNoSessionFound(t *testing.T) {
	mockExtractor := &MockIdentityExtractor{}

	sessionManager := session.NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&session.GeneratorFake{
			SessionIdMock: session.SessionID("mocked-id"),
		},
	)

	manager := &Manager{sessionManager, make(map[session.SessionID]int), sync.Mutex{}}
	mockValidator := &Validator{manager, mockExtractor}
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	mockExtractor := &MockIdentityExtractor{
		identity.FromAddress("wrongsignature"),
		nil,
	}

	mockValidator := &Validator{fakeManager, mockExtractor}

	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsFalseWhenSessionExistsAndSignatureIsValidAndClientIDDiffers(t *testing.T) {
	mockValidator.Validate(1, "not important", "not important")
	authenticated, err := mockValidator.Validate(2, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDMatches(t *testing.T) {
	mockValidator.Validate(1, "not important", "not important")
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}
