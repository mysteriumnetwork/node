package session

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSessionIdLength(t *testing.T) {
	sid := GenerateSessionId()

	assert.Len(t, sid, 36)
}
