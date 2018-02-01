package session

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var generator UUIDGenerator

func TestSessionIdLength(t *testing.T) {
	sid := generator.Generate()

	assert.Len(t, sid, 36)
}
