package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServerInterface(t *testing.T) {
	var _ communication.Server = &serverNats{}
}

func TestNewServer(t *testing.T) {
	address := nats_discovery.NewAddress("127.0.0.1", 4222, "custom")
	server := NewServer(address)

	assert.NotNil(t, server)
	assert.Equal(t, address, server.myAddress)
}
