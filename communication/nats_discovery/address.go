package nats_discovery

import (
	"fmt"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	nats_lib "github.com/nats-io/go-nats"
)

var natsServerIp string

func NewAddress(server string, port int, topic string) *NatsAddress {
	return &NatsAddress{
		servers: []string{
			fmt.Sprintf("nats://%s:%d", server, port),
		},
		topic: topic,
	}
}

func NewBrokerAddresses(contact ContactNATSV1) *NatsAddress {
	return &NatsAddress{
		servers: contact.BrokerAddresses,
		topic: contact.Topic,
	}
}

func NewAddressForIdentity(identity identity.Identity) *NatsAddress {
	return NewAddress(natsServerIp, 4222, identity.Address)
}

func NewAddressForContact(contact dto_discovery.Contact) (*NatsAddress, error) {
	if contact.Type != CONTACT_NATS_V1 {
		return nil, fmt.Errorf("Invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return nil, fmt.Errorf("Invalid contact definition: %#v", contact.Definition)
	}

	return NewBrokerAddresses(contactNats), nil
}

func NewAddressWithConnection(connection nats.Connection, topic string) *NatsAddress {
	return &NatsAddress{
		topic:      topic,
		connection: connection,
	}
}

type NatsAddress struct {
	servers []string
	topic   string

	connection nats.Connection
}

func (address *NatsAddress) Connect() (err error) {
	options := nats_lib.GetDefaultOptions()
	options.Servers = address.servers

	address.connection, err = options.Connect()
	return
}

func (address *NatsAddress) Disconnect() {
	address.connection.Close()
}

func (address *NatsAddress) GetConnection() nats.Connection {
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
			BrokerAddresses: address.servers,
		},
	}
}
