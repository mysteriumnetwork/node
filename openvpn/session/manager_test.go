package session

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManagerAdd(t *testing.T) {
	sessionExpected := session.Session{
		Id:     session.SessionId("mocked-id"),
		Config: "mocked-config",
	}
	manager := manager{
		idGenerator: &session.GeneratorFake{},
	}

	manager.Add(sessionExpected)
	assert.Exactly(
		t,
		[]session.SessionId{sessionExpected.Id},
		manager.sessions,
	)
}

func TestManagerCreate(t *testing.T) {
	sessionExpected := session.Session{
		Id:     session.SessionId("mocked-id"),
		Config: "port 1000\n",
	}

	clientConfig := &openvpn.ClientConfig{&openvpn.Config{}}
	clientConfig.SetPort(1000)

	manager := manager{
		idGenerator: &session.GeneratorFake{
			SessionIdMock: session.SessionId("mocked-id"),
		},
		clientConfig: clientConfig,
	}

	sessionInstance, err := manager.Create()
	assert.NoError(t, err)
	assert.Exactly(t, sessionExpected, sessionInstance)
	assert.Exactly(
		t,
		[]session.SessionId{sessionExpected.Id},
		manager.sessions,
	)
}
