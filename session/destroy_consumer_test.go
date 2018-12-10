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

// managerDestroyFake represents fake destroy Manager usually useful in tests
type managerDestroyFake struct {
	returnSession Session
	returnError   error
}

func TestDestroyConsumer_Success(t *testing.T) {
	mockDestroyer := &managerDestroyFake{
		returnSession: Session{
			"some-session-id",
			fakeSessionConfig{"string-param", 123},
			identity.FromAddress("123"),
		},
	}
	consumer := destroyConsumer{
		SessionDestroyer: mockDestroyer,
		PeerID:           identity.FromAddress("peer-id"),
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
	mockDestroyer := &managerDestroyFake{
		returnError: ErrorSessionNotExists,
	}
	consumer := destroyConsumer{SessionDestroyer: mockDestroyer}

	request := consumer.NewRequest().(*DestroyRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.Error(t, err)
	assert.Exactly(t, destroyResponse(), sessionResponse)
}

func TestDestroyConsumer_ErrorNotSessionOwner(t *testing.T) {
	mockDestroyer := &managerDestroyFake{
		returnError: ErrorWrongSessionOwner,
	}
	consumer := destroyConsumer{SessionDestroyer: mockDestroyer}

	request := consumer.NewRequest().(*DestroyRequest)
	sessionResponse, err := consumer.Consume(request)

	assert.Error(t, err)
	assert.Exactly(t, destroyResponse(), sessionResponse)
}

// Destroy fake destroy function
func (manager *managerDestroyFake) Destroy(consumerID identity.Identity, sessionID string) error {
	return manager.returnError
}
