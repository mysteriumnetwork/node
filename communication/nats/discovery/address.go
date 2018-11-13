/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

import (
	"fmt"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/identity"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	nats_lib "github.com/nats-io/go-nats"
)

// NewAddress creates NATS address to known host or cluster of hosts
func NewAddress(topic string, addresses ...string) *AddressNATS {
	return &AddressNATS{
		servers: addresses,
		topic:   topic,
	}
}

// NewAddressGenerate generates NATS address for current node
func NewAddressGenerate(brokerIP string, myID identity.Identity, serviceType string) *AddressNATS {
	address := fmt.Sprintf("nats://%s:%d", brokerIP, BrokerPort)
	topic := fmt.Sprintf("%v.%v", myID.Address, serviceType)
	return NewAddress(topic, address)
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
	if err != nil {
		address.connection = nil
	}

	return
}

// Disconnect stops currently established connection
func (address *AddressNATS) Disconnect() {
	if address.connection != nil {
		address.connection.Close()
	}
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
