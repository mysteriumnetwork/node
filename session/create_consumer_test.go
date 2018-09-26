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
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var consumer = createConsumer{
	CurrentProposalID: 101,
	SessionManager:    &managerFake{},
}

func TestConsumer_UnknownProposal(t *testing.T) {
	request := consumer.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 100
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: false,
			Message: "Proposal doesn't exist: 100",
		},
		sessionResponse,
	)
}

func TestConsumer_Success(t *testing.T) {
	request := consumer.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 101
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: true,
			Session: SessionDto{
				ID:     "new-id",
				Config: []byte("{\"Param1\":\"string-param\",\"Param2\":123}"),
			},
		},
		sessionResponse,
	)
}

// managerFake represents fake manager usually useful in tests
type managerFake struct{}

var fakeConfig = struct {
	Param1 string
	Param2 int
}{
	"string-param",
	123,
}

// Create function creates and returns fake session
func (manager *managerFake) Create(peerID identity.Identity) (Session, error) {
	return Session{"new-id", fakeConfig, peerID}, nil
}
