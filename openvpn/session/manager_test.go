package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mockedVPNConfig = session.VPNConfig{
	RemoteIP: "1.2.3.4",
}

var expectedSession = session.Session{
	ID:         session.SessionID("mocked-id"),
	Config:     mockedVPNConfig,
	ConsumerID: identity.FromAddress("deadbeef"),
}

type mockedConfigProvider func() session.VPNConfig

func (mcp mockedConfigProvider) ProvideServiceConfig() (session.VPNConfig, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() session.VPNConfig {
	return mockedVPNConfig
}

func TestManagerCreatesNewSession(t *testing.T) {
	manager := NewManager(
		mockedConfigProvider(provideMockedVPNConfig),
		&session.GeneratorFake{
			SessionIdMock: session.SessionID("mocked-id"),
		},
	)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"))
	assert.NoError(t, err)
	assert.Exactly(t, expectedSession, sessionInstance)

	expectedSessionMap := make(map[session.SessionID]session.Session)
	expectedSessionMap[expectedSession.ID] = expectedSession
	assert.Exactly(
		t,
		expectedSessionMap,
		manager.sessionMap,
	)
}

func TestManagerLookupsExistingSession(t *testing.T) {
	sessionMap := make(map[session.SessionID]session.Session)
	sessionMap[expectedSession.ID] = expectedSession

	manager := manager{
		sessionMap: sessionMap,
	}

	session, found := manager.FindSession(session.SessionID("mocked-id"))
	assert.True(t, found)
	assert.Exactly(t, expectedSession, session)
}
