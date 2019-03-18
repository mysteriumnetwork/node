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
	"testing"

	"github.com/mysteriumnetwork/node/nat/traversal"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	currentProposalID = 68
	currentProposal   = market.ServiceProposal{
		ID: currentProposalID,
	}
	consumerID = identity.FromAddress("deadbeef")

	expectedID      = ID("mocked-id")
	expectedSession = Session{
		ID:         expectedID,
		ConsumerID: consumerID,
	}
)

func generateSessionID() (ID, error) {
	return expectedID, nil
}

type mockBalanceTracker struct {
	errorToReturn error
}

func (m mockBalanceTracker) Start() error {
	return m.errorToReturn
}

func (m mockBalanceTracker) Stop() {

}

func mockBalanceTrackerFactory(consumer, provider, issuer identity.Identity) (BalanceTracker, error) {
	return &mockBalanceTracker{}, nil
}

func TestManager_Create_StoresSession(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory()
	natPinger := func(json.RawMessage) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, natPinger, nil, &MockNatEventTracker{})

	requestConfig := json.RawMessage{}
	sessionInstance, err := manager.Create(consumerID, consumerID, currentProposalID, nil, requestConfig)
	expectedResult.Done = sessionInstance.Done
	assert.NoError(t, err)
	assert.Exactly(t, expectedResult, sessionInstance)
}

func TestManager_Create_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewStorageMemory()
	natPinger := func(json.RawMessage) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, natPinger, nil, &MockNatEventTracker{})

	requestConfig := json.RawMessage{}
	sessionInstance, err := manager.Create(consumerID, consumerID, 69, nil, requestConfig)
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Exactly(t, Session{}, sessionInstance)
}

type MockNatEventTracker struct {
}

func (mnet *MockNatEventTracker) LastEvent() traversal.Event {
	return traversal.Event{}
}
