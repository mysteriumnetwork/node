package nats

import (
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	identity = dto_discovery.Identity("123456")
)

func TestNewContact(t *testing.T) {
	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: CONTACT_NATS_V1,
			Definition: ContactNATSV1{
				Topic: string(identity),
			},
		},
		NewContact(identity),
	)
}

func TestNewServer(t *testing.T) {
	var server communication.Server = NewServer()
	assert.NotNil(t, server)
}

func TestNewClient(t *testing.T) {
	client := NewClient(identity)
	assert.NotNil(t, client)
	assert.Equal(t, "123456", client.myTopic)
}
