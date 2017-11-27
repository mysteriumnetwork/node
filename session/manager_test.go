package session

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManagerHasSessionsStored(t *testing.T) {
	var generator GeneratorMock

	manager := Manager{
		Generator: &generator,
	}

	length := 10

	for i := 0; i < length; i++ {
		sid := manager.Generator.Generate()
		manager.Add(sid)
	}

	assert.Len(t, manager.sessions, length)
}
