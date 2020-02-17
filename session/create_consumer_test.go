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
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/stretchr/testify/assert"
)

var (
	config = json.RawMessage(`{"Param1":"string-param","Param2":123}`)
)

func TestConsumer_Success(t *testing.T) {
	mockManager := &managerFake{
		fakeSession: Session{
			ID:         "new-id",
			ConsumerID: identity.FromAddress("123"),
		},
	}
	consumer := createConsumer{
		sessionStarter:         mockManager,
		peerID:                 identity.FromAddress("peer-id"),
		providerConfigProvider: mockConfigProvider{},
	}

	request := consumer.NewRequest().(*CreateRequest)
	request.ProposalID = 101
	request.ConsumerInfo = &ConsumerInfo{
		PaymentVersion: PaymentVersionV3,
	}
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
				Config: config,
			},
			PaymentInfo: PaymentInfo{
				Supports: string(PaymentVersionV3),
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
		sessionStarter:         mockManager,
		providerConfigProvider: mockConfigProvider{},
	}

	request := consumer.NewRequest().(*CreateRequest)
	request.ConsumerInfo = &ConsumerInfo{
		PaymentVersion: PaymentVersionV3,
	}
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, responseInvalidProposal, sessionResponse)
}

func TestConsumer_ErrorFatal(t *testing.T) {
	mockManager := &managerFake{
		returnError: errors.New("fatality"),
	}
	consumer := createConsumer{
		sessionStarter:         mockManager,
		providerConfigProvider: mockConfigProvider{},
	}

	request := consumer.NewRequest().(*CreateRequest)
	request.ConsumerInfo = &ConsumerInfo{
		PaymentVersion: PaymentVersionV3,
	}
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, responseInternalError, sessionResponse)
}

func TestConsumer_UsesIssuerID(t *testing.T) {
	mockManager := &managerFake{
		fakeSession: Session{
			ID:         "new-id",
			ConsumerID: identity.FromAddress("123"),
		},
	}
	consumer := createConsumer{
		sessionStarter:         mockManager,
		peerID:                 identity.FromAddress("peer-id"),
		providerConfigProvider: mockConfigProvider{},
	}

	issuerID := identity.FromAddress("some-peer-id")
	request := consumer.NewRequest().(*CreateRequest)
	request.ProposalID = 101
	request.ConsumerInfo = &ConsumerInfo{
		IssuerID:       issuerID,
		PaymentVersion: PaymentVersionV3,
	}

	_, err := consumer.Consume(request)
	assert.Nil(t, err)
	assert.Equal(t, issuerID, mockManager.lastIssuerID)
}

type mockConfigProvider struct {
}

func (mockConfigProvider) ProvideConfig(_ string, _ json.RawMessage) (*ConfigParams, error) {
	return &ConfigParams{SessionServiceConfig: config, TraversalParams: &traversal.Params{}}, nil
}

// managerFake represents fake Manager usually useful in tests
type managerFake struct {
	lastConsumerID identity.Identity
	lastIssuerID   identity.Identity
	lastProposalID int
	fakeSession    Session
	returnError    error
}

// Start function creates and returns fake session
func (manager *managerFake) Start(session *Session, consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int, config ServiceConfiguration, pingerParams *traversal.Params) error {
	session.ID = manager.fakeSession.ID
	session.ConsumerID = manager.fakeSession.ConsumerID
	manager.lastConsumerID = consumerID
	manager.lastIssuerID = consumerInfo.IssuerID
	manager.lastProposalID = proposalID
	return manager.returnError
}

// Destroy fake destroy function
func (manager *managerFake) Destroy(consumerID identity.Identity, sessionID string) error {
	return nil
}
