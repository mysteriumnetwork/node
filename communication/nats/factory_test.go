package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewService(t *testing.T) {
	_, ok := NewService().(communication.CommunicationsChannel)
	assert.True(t, ok)
}
