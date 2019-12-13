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
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
)

// NewAddress creates NATS address to known host or cluster of hosts
func NewAddress(topic string, connection nats.Connection) *AddressNATS {
	return &AddressNATS{
		topic:      topic,
		connection: connection,
	}
}

// NewAddressForContact extracts NATS address from given contact structure
func NewAddressForContact(contact market.Contact) (*AddressNATS, error) {
	if contact.Type != TypeContactNATSV1 {
		return nil, errors.Errorf("invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(ContactNATSV1)
	if !ok {
		return nil, errors.Errorf("invalid contact definition: %#v", contact.Definition)
	}

	return NewAddress(contactNats.Topic, nats.NewConnection(contactNats.BrokerAddresses...)), nil
}

// AddressNATS structure defines details how NATS ConnectionWrap can be established
type AddressNATS struct {
	topic      string
	connection nats.Connection
}

// GetConnection returns currently established ConnectionWrap
func (address *AddressNATS) GetConnection() nats.Connection {
	return address.connection
}

// GetTopic returns topic.
// Address points to this topic in established ConnectionWrap.
func (address *AddressNATS) GetTopic() string {
	return address.topic
}
