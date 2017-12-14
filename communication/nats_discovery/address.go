package nats_discovery

import (
	"fmt"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
)

func NewAddress(server string, port int, topic string) *NatsAddress {
	return &NatsAddress{
		servers: []string{
			fmt.Sprintf("nats://%s:%d", server, port),
		},
		topic: topic,
	}
}

func NewAddressForIdentity(identity dto_discovery.Identity) *NatsAddress {
	return NewAddress("127.0.0.1", 4222, string(identity))
}

func NewAddressForContact(contact dto_discovery.Contact) (*NatsAddress, error) {
	if contact.Type != CONTACT_NATS_V1 {
		return nil, fmt.Errorf("Invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return nil, fmt.Errorf("Invalid contact definition: %#v", contact.Definition)
	}

	return NewAddress("127.0.0.1", 4222, contactNats.Topic), nil
}

func NewAddressNested(address *NatsAddress, subTopic string) *NatsAddress {
	return &NatsAddress{
		servers:    address.servers,
		topic:      address.topic + "." + subTopic,
		connection: address.connection,
	}
}

type NatsAddress struct {
	servers []string
	topic   string

	connection *nats.Conn
}

func (address *NatsAddress) Connect() (err error) {
	options := nats.GetDefaultOptions()
	options.Servers = address.servers

	address.connection, err = options.Connect()
	return
}

func (address *NatsAddress) Disconnect() {
	address.connection.Close()
}

func (address *NatsAddress) GetConnection() *nats.Conn {
	return address.connection
}

func (address *NatsAddress) GetTopic() string {
	return address.topic
}

func (address *NatsAddress) GetContact() dto_discovery.Contact {
	return dto_discovery.Contact{
		Type: CONTACT_NATS_V1,
		Definition: ContactNATSV1{
			Topic: address.topic,
		},
	}
}
