package nats

import (
	"github.com/magiconair/properties/assert"
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"testing"
)

func TestServerInterface(t *testing.T) {
	var _ communication.Server = &serverNats{}
}

func TestServerGetContact(t *testing.T) {
	identity := dto_discovery.Identity("123456")

	server := &serverNats{
		myIdentity: identity,
	}
	assert.Equal(t, newContact(server.myIdentity), server.GetContact())
}
