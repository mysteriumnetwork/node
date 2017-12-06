package nats

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewContact(t *testing.T) {
	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: "nats/v1",
			Definition: ContactNATSV1{
				Topic: "123456",
			},
		},
		newContact(dto_discovery.Identity("123456")),
	)
}

func TestContactToTopic_UnknownType(t *testing.T) {
	topic, err := contactToTopic(dto_discovery.Contact{
		Type: "natc/v1",
	})
	assert.EqualError(t, err, "Invalid contact type: natc/v1")
	assert.Equal(t, "", topic)
}

func TestContactToTopic_UnknownDefinition(t *testing.T) {
	type badDefinition struct{}

	topic, err := contactToTopic(dto_discovery.Contact{
		Type:       "nats/v1",
		Definition: badDefinition{},
	})
	assert.EqualError(t, err, "Invalid contact definition: nats.badDefinition{}")
	assert.Equal(t, "", topic)
}

func TestContactToTopic_Valid(t *testing.T) {
	topic, err := contactToTopic(dto_discovery.Contact{
		Type: "nats/v1",
		Definition: ContactNATSV1{
			Topic: "123456",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "123456.", topic)
}
