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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/traversal"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
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
		ID:          expectedID,
		ConsumerID:  consumerID,
		ServiceType: "wireguard",
	}
)

type mockBalanceTracker struct {
	errorToReturn error
}

func (m mockBalanceTracker) Start() error {
	return m.errorToReturn
}

func (m mockBalanceTracker) Stop() {

}

func mockPaymentEngineFactory(providerID, consumerID identity.Identity, accountant common.Address, sessionID string) (PaymentEngine, error) {
	return &mockBalanceTracker{}, nil
}

func TestManager_Start_StoresSession(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory()

	manager := newManager(currentProposal, sessionStore)

	pingerParams := &traversal.Params{}
	session, err := NewSession()
	assert.NoError(t, err)
	err = manager.Start(session, consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	assert.NoError(t, err)
	expectedResult.done = session.done

	assert.Equal(t, expectedResult.Config, session.Config)
	assert.Equal(t, expectedResult.Last, session.Last)
	assert.Equal(t, expectedResult.done, session.done)
	assert.Equal(t, expectedResult.ConsumerID, session.ConsumerID)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestManager_Start_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewStorageMemory()

	manager := newManager(currentProposal, sessionStore)

	pingerParams := &traversal.Params{}
	session, err := NewSession()
	assert.NoError(t, err)
	err = manager.Start(session, consumerID, ConsumerInfo{IssuerID: consumerID}, 69, nil, pingerParams)
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Empty(t, session.CreatedAt)
}

type MockNatEventTracker struct {
}

func (mnet *MockNatEventTracker) LastEvent() *event.Event {
	return &event.Event{}
}

func TestManager_AcknowledgeSession_RejectsUnknown(t *testing.T) {
	sessionStore := NewStorageMemory()

	manager := newManager(currentProposal, sessionStore)
	err := manager.Acknowledge(consumerID, "")
	assert.Exactly(t, err, ErrorSessionNotExists)
}

func TestManager_AcknowledgeSession_RejectsBadClient(t *testing.T) {
	sessionStore := NewStorageMemory()

	manager := newManager(currentProposal, sessionStore)

	pingerParams := &traversal.Params{}
	session, err := NewSession()
	err = manager.Start(session, consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	assert.Nil(t, err)

	err = manager.Acknowledge(identity.FromAddress("some other id"), string(session.ID))
	assert.Exactly(t, ErrorWrongSessionOwner, err)
}

func TestManager_AcknowledgeSession_PublishesEvent(t *testing.T) {
	sessionStore := NewStorageMemory()

	mp := mocks.NewEventBus()
	manager := newManager(currentProposal, sessionStore)
	manager.publisher = mp

	pingerParams := &traversal.Params{}
	session, err := NewSession()
	assert.NoError(t, err)
	err = manager.Start(session, consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	assert.Nil(t, err)

	err = manager.Acknowledge(consumerID, string(session.ID))
	assert.Nil(t, err)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Acknowledged), 2*time.Second, 10*time.Millisecond)
}

func newManager(proposal market.ServiceProposal, sessionStore *StorageMemory) *Manager {
	return NewManager(proposal, sessionStore, mockPaymentEngineFactory, traversal.NewNoopPinger(),
		&MockNatEventTracker{}, "test service id", mocks.NewEventBus(), nil, DefaultConfig())
}
