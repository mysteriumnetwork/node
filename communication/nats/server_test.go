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

func TestNewContact(t *testing.T) {
	server := &serverNats{
		myIdentity: dto_discovery.Identity("123456"),
	}

	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: CONTACT_NATS_V1,
			Definition: ContactNATSV1{
				Topic: string(identity),
			},
		},
		server.GetContact(),
	)
}
