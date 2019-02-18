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
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/stretchr/testify/assert"
)

var (
	config       = json.RawMessage(`{"Param1":"string-param","Param2":123}`)
	mockConsumer = func(json.RawMessage) (ServiceConfiguration, DestroyCallback, error) {
		return config, nil, nil
	}
	mockID = identity.FromAddress("0x0")
	errMpl = errors.New("test")
	mpl    = &mockPromiseLoader{
		errorToReturn: errMpl,
	}
)

func TestConsumer_Success(t *testing.T) {
	mockManager := &managerFake{
		returnSession: Session{
			ID:         "new-id",
			ConsumerID: identity.FromAddress("123"),
		},
	}
	consumer := createConsumer{
		sessionCreator: mockManager,
		peerID:         identity.FromAddress("peer-id"),
		configProvider: mockConsumer,
		promiseLoader:  mpl,
	}

	request := consumer.NewRequest().(*CreateRequest)
	request.ProposalID = 101
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
		promiseLoader:  mpl,
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
		promiseLoader:  mpl,
	}

	request := consumer.NewRequest().(*CreateRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(t, responseInternalError, sessionResponse)
}

func TestConsumer_UsesIssuerID(t *testing.T) {
	mockManager := &managerFake{
		returnSession: Session{
			ID:         "new-id",
			ConsumerID: identity.FromAddress("123"),
		},
	}
	consumer := createConsumer{
		sessionCreator: mockManager,
		peerID:         identity.FromAddress("peer-id"),
		configProvider: mockConsumer,
		promiseLoader:  mpl,
	}

	issuerID := identity.FromAddress("some-peer-id")
	request := consumer.NewRequest().(*CreateRequest)
	request.ProposalID = 101
	request.ConsumerInfo = &ConsumerInfo{
		IssuerID: issuerID,
	}

	_, err := consumer.Consume(request)
	assert.Nil(t, err)
	assert.Equal(t, issuerID, mockManager.lastIssuerID)
}

func TestConsumer_loadPaymentInfo_NilOnError(t *testing.T) {
	mockManager := &managerFake{}
	consumer := createConsumer{
		sessionCreator: mockManager,
		configProvider: mockConsumer,
		promiseLoader:  mpl,
	}
	pi := consumer.loadPaymentInfo(mockID)
	assert.Nil(t, pi)
}

func TestConsumer_loadPaymentInfo_ZeroAmountOnNoMessage(t *testing.T) {
	mplCopy := *mpl
	mplCopy.errorToReturn = nil
	mplCopy.promiseToReturn = promise.StoredPromise{
		SequenceID:       10,
		UnconsumedAmount: 0,
	}
	mockManager := &managerFake{}
	consumer := createConsumer{
		sessionCreator: mockManager,
		configProvider: mockConsumer,
		promiseLoader:  &mplCopy,
	}
	pi := consumer.loadPaymentInfo(mockID)
	assert.NotNil(t, pi)
	assert.Equal(t, mplCopy.promiseToReturn.SequenceID, pi.LastPromise.SequenceID)
	assert.Equal(t, uint64(0), pi.LastPromise.Amount)
	assert.Equal(t, mplCopy.promiseToReturn.UnconsumedAmount, pi.FreeCredit)
}

func TestConsumer_loadPaymentInfo_LoadsProperInfo(t *testing.T) {
	mplCopy := *mpl
	mplCopy.errorToReturn = nil
	mplCopy.promiseToReturn = promise.StoredPromise{
		SequenceID:       10,
		UnconsumedAmount: 200,
		Message: &promise.Message{
			Amount:     300,
			SequenceID: 10,
		},
	}
	mockManager := &managerFake{}
	consumer := createConsumer{
		sessionCreator: mockManager,
		configProvider: mockConsumer,
		promiseLoader:  &mplCopy,
	}
	pi := consumer.loadPaymentInfo(mockID)
	assert.NotNil(t, pi)
	assert.Equal(t, mplCopy.promiseToReturn.SequenceID, pi.LastPromise.SequenceID)
	assert.Equal(t, mplCopy.promiseToReturn.Message.Amount, pi.LastPromise.Amount)
	assert.Equal(t, mplCopy.promiseToReturn.UnconsumedAmount, pi.FreeCredit)
}

// managerFake represents fake Manager usually useful in tests
type managerFake struct {
	lastConsumerID identity.Identity
	lastIssuerID   identity.Identity
	lastProposalID int
	returnSession  Session
	returnError    error
}

// Create function creates and returns fake session
func (manager *managerFake) Create(consumerID, issuerID identity.Identity, proposalID int) (Session, error) {
	manager.lastConsumerID = consumerID
	manager.lastIssuerID = issuerID
	manager.lastProposalID = proposalID
	return manager.returnSession, manager.returnError
}

// Destroy fake destroy function
func (manager *managerFake) Destroy(consumerID identity.Identity, sessionID string) error {
	return nil
}

type mockPromiseLoader struct {
	promiseToReturn promise.StoredPromise
	errorToReturn   error
}

func (mpl *mockPromiseLoader) GetLastPromise(identity.Identity) (promise.StoredPromise, error) {
	return mpl.promiseToReturn, mpl.errorToReturn
}
