package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManagerCreatesNewSession(t *testing.T) {
	expectedSession := session.Session{
		ID:         session.SessionID("mocked-id"),
		Config:     "port 1000\n",
		ConsumerID: identity.FromAddress("deadbeef"),
	}

	clientConfig := &openvpn.ClientConfig{&openvpn.Config{}}
	clientConfig.SetPort(1000)

	manager := NewManager(
		clientConfig,
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
	expectedSession := session.Session{
		ID:     session.SessionID("mocked-id"),
		Config: "port 1000\n",
	}

	sessionMap := make(map[session.SessionID]session.Session)
	sessionMap[expectedSession.ID] = expectedSession

	manager := manager{
		sessionMap: sessionMap,
	}

	session, found := manager.FindSession(session.SessionID("mocked-id"))
	assert.True(t, found)
	assert.Exactly(t, expectedSession, session)
}
