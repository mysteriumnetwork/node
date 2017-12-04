package session

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManagerAdd(t *testing.T) {
	sessionExpected := SessionId("session-1")
	manager := Manager{
		Generator: &GeneratorFake{},
	}

	manager.Add(sessionExpected)
	assert.Exactly(
		t,
		[]SessionId{sessionExpected},
		manager.sessions,
	)
}

func TestManagerCreate(t *testing.T) {
	sessionExpected := SessionId("mocked-session")
	manager := Manager{
		Generator: &GeneratorFake{
			SessionIdMock: SessionId("mocked-session"),
		},
	}

	session := manager.Create()
	assert.Exactly(t, sessionExpected, session)
	assert.Exactly(
		t,
		[]SessionId{sessionExpected},
		manager.sessions,
	)
}
