package session

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSessionIdLength(t *testing.T) {
	sid, _ := GenerateSessionId()

	assert.Len(t, sid, 32)
}
