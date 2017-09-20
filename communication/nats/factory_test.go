package nats

import (
	"github.com/mysterium/node/communication"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	contactIdentity = dto_discovery.Identity("123456")
)

func TestNewContact(t *testing.T) {
	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: CONTACT_NATS_V1,
			Definition: ContactNATSV1{
				Topic: string(contactIdentity),
			},
		},
		NewContact(contactIdentity),
	)
}

func TestNewChannel(t *testing.T) {
	var channel communication.Channel = NewChannel()
	assert.NotNil(t, channel)
}
