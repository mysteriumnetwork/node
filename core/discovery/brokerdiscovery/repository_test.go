/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package brokerdiscovery

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	proposalFirst = market.ServiceProposal{
		ProviderID:        "0x1",
		ServiceDefinition: market.UnsupportedServiceDefinition{},
		PaymentMethod:     market.UnsupportedPaymentMethod{},
		ProviderContacts:  []market.Contact{},
	}
	proposalSecond = market.ServiceProposal{
		ProviderID:        "0x2",
		ServiceDefinition: market.UnsupportedServiceDefinition{},
		PaymentMethod:     market.UnsupportedPaymentMethod{},
		ProviderContacts:  []market.Contact{},
	}
)

func Test_Subscriber_StartSyncsNewProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	subscriber := NewRepository(connection, eventbus.New(), 10*time.Millisecond, 1*time.Millisecond)
	err := subscriber.Start()
	defer subscriber.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
		"proposal": {"provider_id": "0x1"}
	}`)
	time.Sleep(5 * time.Millisecond)

	assert.Len(t, subscriber.storage.Proposals(), 1)
	assert.Exactly(t, []market.ServiceProposal{proposalFirst}, subscriber.storage.Proposals())

	proposalRegister(connection, `{
		"proposal": {"provider_id": "0x2"}
	}`)
	time.Sleep(1 * time.Millisecond)

	assert.Len(t, subscriber.storage.Proposals(), 2)
	assert.Exactly(t, []market.ServiceProposal{proposalFirst, proposalSecond}, subscriber.storage.Proposals())
}

func Test_Subscriber_StartSyncsIdleProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	subscriber := NewRepository(connection, eventbus.New(), 10*time.Millisecond, 1*time.Millisecond)
	err := subscriber.Start()
	defer subscriber.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
		"proposal": {"provider_id": "0x1"}
	}`)
	time.Sleep(15 * time.Millisecond)

	assert.Empty(t, subscriber.storage.Proposals())
}

func Test_Subscriber_StartSyncsHealthyProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	subscriber := NewRepository(connection, eventbus.New(), 10*time.Millisecond, 1*time.Millisecond)
	err := subscriber.Start()
	defer subscriber.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
		"proposal": {"provider_id": "0x1"}
	}`)
	time.Sleep(5 * time.Millisecond)

	proposalPing(connection, `{
		"proposal": {"provider_id": "0x1"}
	}`)
	time.Sleep(1 * time.Millisecond)

	timeout := time.After(time.Millisecond * 20)
	for {
		select {
		case <-time.After(time.Millisecond):
			if len(subscriber.storage.Proposals()) == 1 {
				assert.Len(t, subscriber.storage.Proposals(), 1)
				assert.Exactly(t, []market.ServiceProposal{proposalFirst}, subscriber.storage.Proposals())
				return
			}
		case <-timeout:
			assert.Fail(t, "did not get expected proposals before timeout")
			return
		}
	}

}

func Test_Subscriber_StartSyncsStoppedProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	subscriber := NewRepository(connection, eventbus.New(), 10*time.Millisecond, 1*time.Millisecond)
	subscriber.storage.AddProposal(proposalFirst, proposalSecond)
	err := subscriber.Start()
	defer subscriber.Stop()
	assert.NoError(t, err)

	proposalUnregister(connection, `{
		"proposal": {"provider_id": "0x1"}
	}`)
	time.Sleep(1 * time.Millisecond)

	assert.Len(t, subscriber.storage.Proposals(), 1)
	assert.Exactly(t, []market.ServiceProposal{proposalSecond}, subscriber.storage.Proposals())
}

func proposalRegister(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-register", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func proposalUnregister(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-unregister", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func proposalPing(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-ping", []byte(payload))
	if err != nil {
		panic(err)
	}
}
