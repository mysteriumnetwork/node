package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalApiServerPortIsAsExpected(t *testing.T) {

	server, err := CreateNew("", 8000)
	assert.Nil(t, err)

	assert.Equal(t, 8000, server.Port())

	server.Stop()
	server.Wait()
}
