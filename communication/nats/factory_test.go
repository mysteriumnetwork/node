package nats

import (
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
	server := NewServer(identity)
	assert.NotNil(t, server)
	assert.Equal(t, "123456", server.myTopic)
}

func TestNewClient(t *testing.T) {
	client := NewClient(identity)
	assert.NotNil(t, client)
	assert.Equal(t, "123456", client.myTopic)
}
