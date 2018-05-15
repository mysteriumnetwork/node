package session

import (
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mockedVPNConfig = "config_string"

var expectedSession = Session{
	ID:         SessionID("mocked-id"),
	Config:     mockedVPNConfig,
	ConsumerID: identity.FromAddress("deadbeef"),
}

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

func TestManagerCreatesNewSession(t *testing.T) {
	manager := NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&GeneratorFake{
			SessionIdMock: SessionID("mocked-id"),
		},
	)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"))
	assert.NoError(t, err)
	assert.Exactly(t, expectedSession, sessionInstance)

	expectedSessionMap := make(map[SessionID]Session)
	expectedSessionMap[expectedSession.ID] = expectedSession
	assert.Exactly(
		t,
		expectedSessionMap,
		manager.sessionMap,
	)
}

func TestManagerLookupsExistingSession(t *testing.T) {
	sessionMap := make(map[SessionID]Session)
	sessionMap[expectedSession.ID] = expectedSession

	manager := manager{
		sessionMap: sessionMap,
	}

	session, found := manager.FindSession(SessionID("mocked-id"))
	assert.True(t, found)
	assert.Exactly(t, expectedSession, session)
}
