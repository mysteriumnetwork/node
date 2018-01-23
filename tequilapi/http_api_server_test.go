package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalAPIServerPortIsAsExpected(t *testing.T) {
	server := NewServer("localhost", 31337, nil)

	assert.NoError(t, server.StartServing())

	port, err := server.Port()
	assert.NoError(t, err)
	assert.Equal(t, 31337, port)

	server.Stop()
	server.Wait()
}

func TestStopBeforeStartingListeningDoesNotCausePanic(t *testing.T) {
	server := NewServer("", 12345, nil)
	server.Stop()
}
