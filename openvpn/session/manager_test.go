package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

func NewFakeClientConfigGenerator(port int) openvpn.ClientConfigGenerator {
	return func() *openvpn.ClientConfig {
		vpnClientConfig := &openvpn.ClientConfig{&openvpn.Config{}}
		vpnClientConfig.SetPort(port)
		return vpnClientConfig
	}
}

func TestManagerCreatesNewSession(t *testing.T) {
	expectedSession := session.Session{
		ID:         session.SessionID("mocked-id"),
		Config:     "port 1000\n",
		ConsumerID: identity.FromAddress("deadbeef"),
	}

	clientConfigGenerator := NewFakeClientConfigGenerator(1000)

	manager := NewManager(
		clientConfigGenerator,
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
