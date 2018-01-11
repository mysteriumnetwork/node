package discovery

import (
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAddress(t *testing.T) {
	address := NewAddress("topic1234", "nats://far-server:1234")

	assert.Equal(
		t,
		&NatsAddress{
			servers: []string{"nats://far-server:1234"},
			topic:   "topic1234",
		},
		address,
	)
}

func TestNewAddressGenerate(t *testing.T) {
	myIdentity := identity.FromAddress("provider1")
	address := NewAddressGenerate(myIdentity)

	assert.Equal(
		t,
		&NatsAddress{
			servers: []string{"nats://" + natsServerIp + ":4222"},
			topic:   "provider1",
		},
		address,
	)
}

func TestNewAddressForContact(t *testing.T) {
	address, err := NewAddressForContact(dto_discovery.Contact{
		Type: "nats/v1",
		Definition: ContactNATSV1{
			Topic:           "123456",
			BrokerAddresses: []string{"nats://far-server:4222"},
		},
	})

	assert.NoError(t, err)
	assert.Equal(
		t,
		&NatsAddress{
			servers: []string{"nats://far-server:4222"},
			topic:   "123456",
		},
		address,
	)
}

func TestNewAddressForContact_UnknownType(t *testing.T) {
	address, err := NewAddressForContact(dto_discovery.Contact{
		Type: "natc/v1",
	})

	assert.EqualError(t, err, "Invalid contact type: natc/v1")
	assert.Nil(t, address)
}

func TestNewAddressForContact_UnknownDefinition(t *testing.T) {
	type badDefinition struct{}

	address, err := NewAddressForContact(dto_discovery.Contact{
		Type:       "nats/v1",
		Definition: badDefinition{},
	})

	assert.EqualError(t, err, "Invalid contact definition: discovery.badDefinition{}")
	assert.Nil(t, address)
}

func TestAddress_GetConnection(t *testing.T) {
	expectedConnectin := &nats.Conn{}
	address := &NatsAddress{connection: expectedConnectin}

	assert.Exactly(t, expectedConnectin, address.GetConnection())
}

func TestAddress_GetTopic(t *testing.T) {
	address := &NatsAddress{topic: "123456"}

	assert.Equal(t, "123456", address.GetTopic())
}

func TestAddress_GetContact(t *testing.T) {
	address := &NatsAddress{
		servers: []string{"nats://far-server:4222"},
		topic:   "123456",
	}

	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: "nats/v1",
			Definition: ContactNATSV1{
				Topic:           "123456",
				BrokerAddresses: []string{"nats://far-server:4222"},
			},
		},
		address.GetContact(),
	)
}
