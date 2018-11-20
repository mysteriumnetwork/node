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

	"github.com/mysteriumnetwork/node/identity"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
)

func TestNewAddress(t *testing.T) {
	address := NewAddress("topic1234", "nats://far-server:1234")

	assert.Equal(
		t,
		&AddressNATS{
			servers: []string{"nats://far-server:1234"},
			topic:   "topic1234",
		},
		address,
	)
}

func TestNewAddressFromHostAndID(t *testing.T) {
	var tests = []struct {
		uri  string
		want string
	}{
		{"127.0.0.1", "nats://127.0.0.1:4222"},
		{"nats://127.0.0.1", "nats://127.0.0.1:4222"},
		{"127.0.0.1:4222", "nats://127.0.0.1:4222"},
		{"nats://127.0.0.1:4222", "nats://127.0.0.1:4222"},

		{"nats://127.0.0.1:4333", "nats://127.0.0.1:4333"},
		{"nats://example.com:4333", "nats://example.com:4333"},
	}

	myID := identity.FromAddress("provider1")
	for _, tc := range tests {
		address, err := NewAddressFromHostAndID(tc.uri, myID, "noop")
		assert.NoError(t, err)
		assert.Equal(
			t,
			&AddressNATS{
				servers: []string{tc.want},
				topic:   "provider1.noop",
			},
			address,
		)

	}
}

func TestNewAddressForContact(t *testing.T) {
	address, err := NewAddressForContact(dto_discovery.Contact{
		Type: "nats/v1",
		Definition: ContactNATSV1{
			Topic:           "123456",
			BrokerAddresses: []string{"nats://far-server:4222"},
		},
	})

	assert.NoError(t, err)
	assert.Equal(
		t,
		&AddressNATS{
			servers: []string{"nats://far-server:4222"},
			topic:   "123456",
		},
		address,
	)
}

func TestNewAddressForContact_UnknownType(t *testing.T) {
	address, err := NewAddressForContact(dto_discovery.Contact{
		Type: "natc/v1",
	})

	assert.EqualError(t, err, "invalid contact type: natc/v1")
	assert.Nil(t, address)
}

func TestNewAddressForContact_UnknownDefinition(t *testing.T) {
	type badDefinition struct{}

	address, err := NewAddressForContact(dto_discovery.Contact{
		Type:       "nats/v1",
		Definition: badDefinition{},
	})

	assert.EqualError(t, err, "invalid contact definition: discovery.badDefinition{}")
	assert.Nil(t, address)
}

func TestAddress_Disconnect_EmptyAddress(t *testing.T) {
	address := &AddressNATS{}
	address.Disconnect()
}

func TestAddress_Disconnect_AfterFailedConnect(t *testing.T) {
	address := &AddressNATS{
		servers: []string{"nats://far-server:4222"},
	}

	assert.EqualError(t, address.Connect(), "nats: no servers available for connection")
	address.Disconnect()
}

func TestAddress_GetConnection(t *testing.T) {
	expectedConnectin := &nats.Conn{}
	address := &AddressNATS{connection: expectedConnectin}

	assert.Exactly(t, expectedConnectin, address.GetConnection())
}

func TestAddress_GetTopic(t *testing.T) {
	address := &AddressNATS{topic: "123456"}

	assert.Equal(t, "123456", address.GetTopic())
}

func TestAddress_GetContact(t *testing.T) {
	address := &AddressNATS{
		servers: []string{"nats://far-server:4222"},
		topic:   "123456",
	}

	assert.Equal(
		t,
		dto_discovery.Contact{
			Type: "nats/v1",
			Definition: ContactNATSV1{
				Topic:           "123456",
				BrokerAddresses: []string{"nats://far-server:4222"},
			},
		},
		address.GetContact(),
	)
}
