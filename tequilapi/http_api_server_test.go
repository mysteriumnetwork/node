package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalApiServerPortIsAsExpected(t *testing.T) {
	server, err := StartNewServer("", 31337, nil)
	assert.Nil(t, err)

	assert.Equal(t, 31337, server.Port())

	server.Stop()
	server.Wait()
}
