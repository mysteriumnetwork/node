package session

import (
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManagerAdd(t *testing.T) {
	sessionExpected := session.Session{
		Id:     session.SessionId("mocked-session"),
		Config: "mocked-config",
	}
	manager := Manager{
		Generator: &session.GeneratorFake{},
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
		Id:     session.SessionId("mocked-session"),
		Config: "",
	}
	manager := Manager{
		Generator: &session.GeneratorFake{
			SessionIdMock: session.SessionId("mocked-session"),
		},
	}

	sessionInstance := manager.Create()
	assert.Exactly(t, sessionExpected, sessionInstance)
	assert.Exactly(
		t,
		[]session.SessionId{sessionExpected.Id},
		manager.sessions,
	)
}
