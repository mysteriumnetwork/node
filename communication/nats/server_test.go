package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServerInterface(t *testing.T) {
	var _ communication.Server = &serverNats{}
}

func TestNewServer(t *testing.T) {
	identity := dto_discovery.Identity("123456")
	server := NewServer(identity)

	assert.NotNil(t, server)
	assert.Equal(t, nats_discovery.NewAddressForIdentity(identity), server.myAddress)
}
