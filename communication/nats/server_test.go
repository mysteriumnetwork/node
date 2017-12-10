package nats

import (
	"github.com/magiconair/properties/assert"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	"testing"
)

func TestServerInterface(t *testing.T) {
	var _ communication.Server = &serverNats{}
}

func TestServerGetContact(t *testing.T) {
	address := nats_discovery.NewAddress("far-server", 1234, "custom")

	server := &serverNats{
		myAddress: address,
	}
	assert.Equal(t, address.GetContact(), server.GetContact())
}
