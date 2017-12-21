package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalApiServerPortIsAsExpected(t *testing.T) {
	server, err := NewServer("localhost", 31337, nil)
	assert.Nil(t, err)
	server.StartServing()
	assert.Equal(t, 31337, server.Port())

	server.Stop()
	server.Wait()
}
