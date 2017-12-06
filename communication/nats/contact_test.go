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
			Type: CONTACT_NATS_V1,
			Definition: ContactNATSV1{
				Topic: "123456",
			},
		},
		newContact(dto_discovery.Identity("123456")),
	)
}
