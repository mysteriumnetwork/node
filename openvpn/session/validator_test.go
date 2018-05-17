package session

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/identity"
	"sync"
)

type mockSessionManager struct {
	onFindReturnSession session.Session
	onFindReturnSuccess bool
}

func (manager *mockSessionManager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return session.Session{}, nil
}

func (manager *mockSessionManager) FindSession(sessionId session.SessionID) (session.Session, bool) {
	return manager.onFindReturnSession, manager.onFindReturnSuccess
}

type mockIdentityExtractor struct {
	onExtractReturnIdentity identity.Identity
	onExtractReturnError    error
}

func (extractor *mockIdentityExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.onExtractReturnIdentity, extractor.onExtractReturnError
}

var mockedVPNConfig = "config_string"

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

var mockManager = &mockSessionManager{
	session.Session{
		ID:         session.SessionID("fake-id"),
		Config:     mockedVPNConfig,
		ConsumerID: identity.FromAddress("deadbeef"),
	},
	true,
}

var mockExtractor = &mockIdentityExtractor{
	identity.FromAddress("deadbeef"),
	nil,
}

var fakeManager = &manager{
	mockManager,
	make(map[session.SessionID]int),
	sync.Mutex{},
}

var mockValidator = &Validator{
	fakeManager,
	mockExtractor,
}

func TestValidateReturnsFalseWhenNoSessionFound(t *testing.T) {
	mockExtractor := &mockIdentityExtractor{}

	sessionManager := session.NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&session.GeneratorFake{
			SessionIdMock: session.SessionID("mocked-id"),
		},
	)

	manager := &manager{sessionManager, make(map[session.SessionID]int), sync.Mutex{}}
	mockValidator := &Validator{manager, mockExtractor}
	authenticated, err := mockValidator.Validate(1, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	mockExtractor := &mockIdentityExtractor{
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
