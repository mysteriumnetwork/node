package discovery

import (
	"fmt"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	nats_lib "github.com/nats-io/go-nats"
	"strconv"
)

// NewAddress creates NATS address to known host or cluster of hosts
func NewAddress(topic string, addresses ...string) *AddressNATS {
	return &AddressNATS{
		servers: addresses,
		topic:   topic,
	}
}

// NewAddressGenerate generates NATS address for current node
func NewAddressGenerate(brokerIP string, myID identity.Identity) *AddressNATS {
	address := "nats://" + brokerIP + ":" + strconv.Itoa(BrokerPort)
	return NewAddress(myID.Address, address)
}

// NewAddressForContact extracts NATS address from given contact structure
func NewAddressForContact(contact dto_discovery.Contact) (*AddressNATS, error) {
	if contact.Type != TypeContactNATSV1 {
		return nil, fmt.Errorf("invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return nil, fmt.Errorf("invalid contact definition: %#v", contact.Definition)
	}

	return &AddressNATS{
		servers: contactNats.BrokerAddresses,
		topic:   contactNats.Topic,
	}, nil
}

// NewAddressWithConnection constructs NATS address to already active NATS connection
func NewAddressWithConnection(connection nats.Connection, topic string) *AddressNATS {
	return &AddressNATS{
		topic:      topic,
		connection: connection,
	}
}

// AddressNATS structure defines details how NATS connection can be established
type AddressNATS struct {
	servers []string
	topic   string

	connection nats.Connection
}

// Connect establishes connection to broker
func (address *AddressNATS) Connect() (err error) {
	options := nats_lib.GetDefaultOptions()
	options.Servers = address.servers
	options.MaxReconnect = BrokerMaxReconnect
	options.ReconnectWait = BrokerReconnectWait
	options.Timeout = BrokerTimeout

	address.connection, err = options.Connect()
	return
}

// Disconnect stops currently established connection
func (address *AddressNATS) Disconnect() {
	address.connection.Close()
}

// GetConnection returns currently established connection
func (address *AddressNATS) GetConnection() nats.Connection {
	return address.connection
}

// GetTopic returns topic.
// Address points to this topic in established connection.
func (address *AddressNATS) GetTopic() string {
	return address.topic
}

// GetContact serializes current address to Contact structure.
func (address *AddressNATS) GetContact() dto_discovery.Contact {
	return dto_discovery.Contact{
		Type: TypeContactNATSV1,
		Definition: ContactNATSV1{
			Topic:           address.topic,
			BrokerAddresses: address.servers,
		},
	}
}
