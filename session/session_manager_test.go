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
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

var (
	currentProposalID = 68
	currentProposal   = market.ServiceProposal{
		ID: currentProposalID,
	}
	consumerID   = identity.FromAddress("deadbeef")
	accountantID = common.HexToAddress("0x1")

	expectedID      = ID("mocked-id")
	expectedSession = Session{
		ID:         expectedID,
		ConsumerID: consumerID,
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

func (m mockBalanceTracker) WaitFirstInvoice(time.Duration) error {
	return nil
}

func mockPaymentEngineFactory(providerID, consumerID identity.Identity, accountant common.Address, sessionID string) (PaymentEngine, error) {
	return &mockBalanceTracker{}, nil
}

func TestManager_Start_StoresSession(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory(mocks.NewEventBus())

	manager := newManager(currentProposal, sessionStore)

	session, err := manager.Start(consumerID, accountantID, currentProposalID)
	assert.NoError(t, err)
	expectedResult.done = session.done

	assert.Equal(t, expectedResult.done, session.done)
	assert.Equal(t, expectedResult.ConsumerID, session.ConsumerID)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestManager_Start_Second_Session_Destroy_Stale_Session(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory(mocks.NewEventBus())

	manager := newManager(currentProposal, sessionStore)

	session1, err := manager.Start(consumerID, accountantID, currentProposalID)
	assert.NoError(t, err)
	expectedResult.done = session1.done
	_, found := sessionStore.Find(session1.ID)

	assert.Equal(t, expectedResult.done, session1.done)
	assert.Equal(t, expectedResult.ConsumerID, session1.ConsumerID)
	assert.False(t, session1.CreatedAt.IsZero())
	assert.True(t, found)

	session2, err := manager.Start(consumerID, accountantID, currentProposalID)
	assert.NoError(t, err)
	expectedResult.done = session2.done

	assert.Equal(t, expectedResult.done, session2.done)
	assert.Equal(t, expectedResult.ConsumerID, session2.ConsumerID)
	assert.False(t, session2.CreatedAt.IsZero())
	assert.Eventuallyf(t, func() bool {
		_, found = sessionStore.Find(session1.ID)
		return !found
	}, time.Second, 10*time.Millisecond, "Waiting for session destroy")
}

func TestManager_Start_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewStorageMemory(mocks.NewEventBus())

	manager := newManager(currentProposal, sessionStore)

	session, err := manager.Start(consumerID, accountantID, 69)
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Empty(t, session.CreatedAt)
}

type MockNatEventTracker struct {
}

func (mnet *MockNatEventTracker) LastEvent() *event.Event {
	return &event.Event{}
}

func TestManager_AcknowledgeSession_RejectsUnknown(t *testing.T) {
	sessionStore := NewStorageMemory(mocks.NewEventBus())

	manager := newManager(currentProposal, sessionStore)
	err := manager.Acknowledge(consumerID, "")
	assert.Exactly(t, err, ErrorSessionNotExists)
}

func TestManager_AcknowledgeSession_RejectsBadClient(t *testing.T) {
	sessionStore := NewStorageMemory(mocks.NewEventBus())

	manager := newManager(currentProposal, sessionStore)

	session, err := manager.Start(consumerID, accountantID, currentProposalID)
	assert.Nil(t, err)

	err = manager.Acknowledge(identity.FromAddress("some other id"), string(session.ID))
	assert.Exactly(t, ErrorWrongSessionOwner, err)
}

func TestManager_AcknowledgeSession_PublishesEvent(t *testing.T) {
	sessionStore := NewStorageMemory(mocks.NewEventBus())

	mp := mocks.NewEventBus()
	manager := newManager(currentProposal, sessionStore)
	manager.publisher = mp

	session, err := manager.Start(consumerID, accountantID, currentProposalID)
	assert.Nil(t, err)

	err = manager.Acknowledge(consumerID, string(session.ID))
	assert.Nil(t, err)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.AcknowledgedStatus), 2*time.Second, 10*time.Millisecond)
}

func newManager(proposal market.ServiceProposal, sessionStore *StorageMemory) *SessionManager {
	return NewSessionManager(proposal, sessionStore, mockPaymentEngineFactory, &MockNatEventTracker{}, "test service id", mocks.NewEventBus(), nil, DefaultConfig())
}
