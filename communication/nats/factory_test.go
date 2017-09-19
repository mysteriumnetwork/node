package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewChannel(t *testing.T) {
	_, ok := NewChannel().(communication.Channel)
	assert.True(t, ok)
}
