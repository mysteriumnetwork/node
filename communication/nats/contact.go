package nats

import (
	"fmt"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

const CONTACT_NATS_V1 = "nats/v1"

type ContactNATSV1 struct {
	// Topic on which client is getting message
	Topic string
}

func newContact(identity dto_discovery.Identity) dto_discovery.Contact {
	return dto_discovery.Contact{
		Type: CONTACT_NATS_V1,
		Definition: ContactNATSV1{
			Topic: identityToTopic(identity),
		},
	}
}

func contactToTopic(contact dto_discovery.Contact) (string, error) {
	if contact.Type != CONTACT_NATS_V1 {
		return "", fmt.Errorf("Invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return "", fmt.Errorf("Invalid contact definition: %#v", contact.Definition)
	}

	return contactNats.Topic + ".", nil
}

func identityToTopic(identity dto_discovery.Identity) string {
	return string(identity)
}
