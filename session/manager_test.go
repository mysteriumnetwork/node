package session

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestManagerHasUniqueSessionsStored(t *testing.T) {
	manager := Manager{}
	length := 10

	for i := 0; i < length; i++ {
		manager.Create()
	}

	assert.Len(t, manager.Sessions, length)
}
