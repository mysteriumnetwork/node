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

package session

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var (
	mockConsumer = func(json.RawMessage) (ServiceConfiguration, error) {
		return nil, nil
	}
)

func TestConsumer_Success(t *testing.T) {
	mockManager := &managerFake{
		returnSession: Session{
			ID:         "new-id",
			Config:     fakeSessionConfig{"string-param", 123},
			ConsumerID: identity.FromAddress("123"),
		},
	}
	consumer := createConsumer{
		sessionCreator: mockManager,
		peerID:         identity.FromAddress("peer-id"),
		configProvider: mockConsumer,
	}

	request := consumer.NewRequest().(*CreateRequest)
	request.ProposalId = 101
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, mockManager.lastConsumerID, identity.FromAddress("peer-id"))
	assert.Exactly(t, mockManager.lastProposalID, 101)
	assert.Exactly(
		t,
		CreateResponse{
			Success: true,
			Session: SessionDto{
				ID:     "new-id",
				Config: []byte(`{"Param1":"string-param","Param2":123}`),
			},
		},
		sessionResponse,
	)
}

func TestConsumer_ErrorInvalidProposal(t *testing.T) {
	mockManager := &managerFake{
		returnError: ErrorInvalidProposal,
	}
	consumer := createConsumer{
		sessionCreator: mockManager,
		configProvider: mockConsumer,
	}

	request := consumer.NewRequest().(*CreateRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, responseInvalidProposal, sessionResponse)
}

func TestConsumer_ErrorFatal(t *testing.T) {
	mockManager := &managerFake{
		returnError: errors.New("fatality"),
	}
	consumer := createConsumer{
		sessionCreator: mockManager,
		configProvider: mockConsumer,
	}

	request := consumer.NewRequest().(*CreateRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, responseInternalError, sessionResponse)
}

// managerFake represents fake Manager usually useful in tests
type managerFake struct {
	lastConsumerID identity.Identity
	lastProposalID int
	returnSession  Session
	returnError    error
}

// Create function creates and returns fake session
func (manager *managerFake) Create(consumerID identity.Identity, proposalID int, config ServiceConfiguration) (Session, error) {
	manager.lastConsumerID = consumerID
	manager.lastProposalID = proposalID
	return manager.returnSession, manager.returnError
}

// Destroy fake destroy function
func (manager *managerFake) Destroy(consumerID identity.Identity, sessionID string) error {
	return nil
}
