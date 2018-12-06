/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

// managerFake represents fake manager usually useful in tests
type managerDestroyFake struct {
	lastConsumerID identity.Identity
	lastProposalID int
	returnSession  Session
	returnError    error
}

func TestDestroyConsumer_Success(t *testing.T) {
	mockManager := &managerDestroyFake{
		returnSession: Session{
			"some-session-id",
			fakeSessionConfig{"string-param", 123},
			identity.FromAddress("123"),
		},
	}
	consumer := destroyConsumer{
		SessionManager: mockManager,
		PeerID:         identity.FromAddress("peer-id"),
	}

	request := consumer.NewRequest().(*DestroyRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		DestroyResponse{
			Success: true,
		},
		sessionResponse,
	)
}

func TestDestroyConsumer_ErrorInvalidSession(t *testing.T) {
	mockManager := &managerDestroyFake{
		returnError: ErrorSessionNotExists,
	}
	consumer := destroyConsumer{SessionManager: mockManager}

	request := consumer.NewRequest().(*DestroyRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.Error(t, err)
	assert.Exactly(t, destroyResponse(ErrorSessionNotExists), sessionResponse)
}

func TestDestroyConsumer_ErrorNotSessionOwner(t *testing.T) {
	mockManager := &managerDestroyFake{
		returnError: ErrorWrongSessionOwner,
	}
	consumer := destroyConsumer{SessionManager: mockManager}

	request := consumer.NewRequest().(*DestroyRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.Error(t, err)
	assert.Exactly(t, destroyResponse(ErrorWrongSessionOwner), sessionResponse)
}

// Create function creates and returns fake session
func (manager *managerDestroyFake) Create(consumerID identity.Identity, proposalID int) (Session, error) {
	return Session{}, nil
}

func (manager *managerDestroyFake) Destroy(consumerID identity.Identity, serviceID string) error {
	return manager.returnError
}
