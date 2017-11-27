package session

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var generator Generator

func TestSessionIdLength(t *testing.T) {
	sid := generator.Generate()

	assert.Len(t, sid, 36)
}
