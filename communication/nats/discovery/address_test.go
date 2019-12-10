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
	"testing"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

func TestNewAddress(t *testing.T) {
	connection := nats.NewConnection("nats://far-server:1234")
	address := NewAddress("topic1234", connection)

	assert.Equal(
		t,
		&AddressNATS{
			topic:      "topic1234",
			connection: connection,
		},
		address,
	)
}

func TestNewAddressForContact(t *testing.T) {
	address, err := NewAddressForContact(market.Contact{
		Type: "nats/v1",
		Definition: ContactNATSV1{
			Topic:           "123456",
			BrokerAddresses: []string{"nats://far-server:4222"},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "123456", address.topic)
	assert.Equal(t, []string{"nats://far-server:4222"}, address.connection.Servers())
}

func TestNewAddressForContact_UnknownType(t *testing.T) {
	address, err := NewAddressForContact(market.Contact{
		Type: "natc/v1",
	})

	assert.EqualError(t, err, "invalid contact type: natc/v1")
	assert.Nil(t, address)
}

func TestNewAddressForContact_UnknownDefinition(t *testing.T) {
	type badDefinition struct{}

	address, err := NewAddressForContact(market.Contact{
		Type:       "nats/v1",
		Definition: badDefinition{},
	})

	assert.EqualError(t, err, "invalid contact definition: discovery.badDefinition{}")
	assert.Nil(t, address)
}

func TestAddress_GetConnection(t *testing.T) {
	expectedConnection := &nats.ConnectionWrap{}
	address := &AddressNATS{connection: expectedConnection}

	assert.Exactly(t, expectedConnection, address.GetConnection())
}

func TestAddress_GetTopic(t *testing.T) {
	address := &AddressNATS{topic: "123456"}

	assert.Equal(t, "123456", address.GetTopic())
}

func TestAddress_GetContact(t *testing.T) {
	address := &AddressNATS{
		topic:      "123456",
		connection: nats.NewConnection("nats://far-server:4222"),
	}

	assert.Equal(
		t,
		market.Contact{
			Type: "nats/v1",
			Definition: ContactNATSV1{
				Topic:           "123456",
				BrokerAddresses: []string{"nats://far-server:4222"},
			},
		},
		address.GetContact(),
	)
}
