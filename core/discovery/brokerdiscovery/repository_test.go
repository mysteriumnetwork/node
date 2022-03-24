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
	"encoding/json"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

func init() {
	market.RegisterServiceType("mock_service")
	market.RegisterContactUnserializer("mock_contact",
		func(rawMessage *json.RawMessage) (market.ContactDefinition, error) {
			return mockContact{}, nil
		},
	)
}

var (
	proposalFirst = func() market.ServiceProposal {
		return market.NewProposal("0x1", "mock_service", market.NewProposalOpts{
			Contacts: []market.Contact{{Type: "mock_contact", Definition: mockContact{}}},
		})
	}
	proposalSecond = func() market.ServiceProposal {
		return market.NewProposal("0x2", "mock_service", market.NewProposalOpts{
			Contacts: []market.Contact{{Type: "mock_contact", Definition: mockContact{}}},
		})
	}
)

func Test_Subscriber_StartSyncsNewProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	repo := NewRepository(connection, NewStorage(eventbus.New()), 500*time.Millisecond, 1*time.Second)
	err := repo.Start()
	defer repo.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `
		{
		  "proposal": {
			"format": "service-proposal/v3",
			"compatibility": 2,
			"provider_id": "0x1",
			"service_type": "mock_service",
			"contacts": [
			  {
				"type": "mock_contact"
			  }
			]
		  }
		}
	`)

	assert.Eventually(t, proposalCountEquals(repo, 1), 2*time.Second, 10*time.Millisecond)
	assert.Exactly(t, []market.ServiceProposal{proposalFirst()}, repo.storage.Proposals())
}

func Test_Subscriber_SkipUnsupportedProposal(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	repo := NewRepository(connection, NewStorage(eventbus.New()), 500*time.Millisecond, 10*time.Millisecond)
	err := repo.Start()
	defer repo.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
		"proposal": {"provider_id": "0x1", "service_type": "unknown"}
	}`)

	time.Sleep(10 * time.Millisecond)
	assert.Len(t, repo.storage.Proposals(), 0)
	assert.Exactly(t, []market.ServiceProposal{}, repo.storage.Proposals())
}

func Test_Subscriber_StartSyncsIdleProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	repo := NewRepository(connection, NewStorage(eventbus.New()), 10*time.Millisecond, 10*time.Millisecond)
	err := repo.Start()
	defer repo.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
	  "proposal": {
		"format": "service-proposal/v2",
		"provider_id": "0x1",
		"service_type": "mock_service",
		"contacts": [
		  {
			"type": "mock_contact"
		  }
		]
	  }
	}`)
	assert.Eventually(t, proposalCountEquals(repo, 0), 2*time.Second, 10*time.Millisecond)
}

func Test_Subscriber_StartSyncsHealthyProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	repo := NewRepository(connection, NewStorage(eventbus.New()), 100*time.Millisecond, 10*time.Millisecond)
	err := repo.Start()
	defer repo.Stop()
	assert.NoError(t, err)

	proposalRegister(connection, `{
	  "proposal": {
		"format": "service-proposal/v3",
		"compatibility": 2,
		"provider_id": "0x1",
		"service_type": "mock_service",
		"contacts": [
		  {
			"type": "mock_contact"
		  }
		]
	  }
	}`)

	proposalPing(connection, `{
	  "proposal": {
        "format": "service-proposal/v3",
		"compatibility": 2,
		"provider_id": "0x1",
		"service_type": "mock_service",
		"contacts": [
		  {
			"type": "mock_contact"
		  }
		]
	  }
	}`)

	assert.Eventually(t, proposalCountEquals(repo, 1), 2*time.Second, 10*time.Millisecond)
	expected := []market.ServiceProposal{proposalFirst()}
	actual := repo.storage.Proposals()
	assert.Exactly(t, expected, actual)
}

func Test_Subscriber_StartSyncsStoppedProposals(t *testing.T) {
	connection := nats.StartConnectionMock()
	defer connection.Close()

	repo := NewRepository(connection, NewStorage(eventbus.New()), 500*time.Millisecond, 10*time.Millisecond)
	repo.storage.AddProposal(proposalFirst(), proposalSecond())
	err := repo.Start()
	defer repo.Stop()
	assert.NoError(t, err)

	proposalUnregister(connection, `{
	  "proposal": {
		"format": "service-proposal/v2",
		"provider_id": "0x1",
		"service_type": "mock_service",
		"contacts": [
		  {
			"type": "mock_contact"
		  }
		]
	  }
}`)
}

func proposalRegister(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-register.v3", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func proposalUnregister(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-unregister.v3", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func proposalPing(connection nats.Connection, payload string) {
	err := connection.Publish("*.proposal-ping.v3", []byte(payload))
	if err != nil {
		panic(err)
	}
}

func proposalCountEquals(subscriber *Repository, count int) func() bool {
	return func() bool {
		return len(subscriber.storage.Proposals()) == count
	}
}

type mockContact struct{}
