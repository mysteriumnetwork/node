package nats

import (
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	identity = dto_discovery.Identity("123456")
)

func TestNewServer(t *testing.T) {
	server := NewServer(identity)

	assert.NotNil(t, server)
	assert.Equal(t, nats_discovery.NewAddressForIdentity(identity), server.myAddress)
}

func TestNewClient(t *testing.T) {
	client := NewClient(identity)

	assert.NotNil(t, client)
	assert.Equal(t, identity, client.myIdentity)
}
