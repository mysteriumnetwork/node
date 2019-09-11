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
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
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
		ID:         expectedID,
		ConsumerID: consumerID,
	}
)

type mockPublisher struct {
	published sessionEvent.Payload
	lock      sync.Mutex
}

func (mp *mockPublisher) Publish(topic string, data interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	mp.published = data.(sessionEvent.Payload)
}

func (mp *mockPublisher) SubscribeAsync(topic string, f interface{}) error {
	return nil
}

func (mp *mockPublisher) getLast() sessionEvent.Payload {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	return mp.published
}

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

func mockBalanceTrackerFactory(consumer, provider, issuer identity.Identity) (PaymentEngine, error) {
	return &mockBalanceTracker{}, nil
}

func mockPaymentEngineFactory() (PaymentEngine, error) {
	return &mockBalanceTracker{}, nil
}

func TestManager_Create_StoresSession(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory()

	natPinger := func(*traversal.Params) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, mockPaymentEngineFactory, natPinger,
		&MockNatEventTracker{}, "test service id", &mockPublisher{})

	pingerParams := &traversal.Params{}
	sessionInstance, err := manager.Create(consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	expectedResult.done = sessionInstance.done
	assert.NoError(t, err)

	assert.Equal(t, expectedResult.Config, sessionInstance.Config)
	assert.Equal(t, expectedResult.Last, sessionInstance.Last)
	assert.Equal(t, expectedResult.done, sessionInstance.done)
	assert.Equal(t, expectedResult.ConsumerID, sessionInstance.ConsumerID)
	assert.Equal(t, expectedResult.ID, sessionInstance.ID)
	assert.False(t, sessionInstance.CreatedAt.IsZero())
}

func TestManager_Create_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewStorageMemory()
	natPinger := func(*traversal.Params) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, mockPaymentEngineFactory, natPinger,
		&MockNatEventTracker{}, "test service id", &mockPublisher{})

	pingerParams := &traversal.Params{}
	sessionInstance, err := manager.Create(consumerID, ConsumerInfo{IssuerID: consumerID}, 69, nil, pingerParams)
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Exactly(t, Session{}, sessionInstance)
}

type MockNatEventTracker struct {
}

func (mnet *MockNatEventTracker) LastEvent() *event.Event {
	return &event.Event{}
}

func TestManager_AcknowledgeSession_RejectsUnknown(t *testing.T) {
	sessionStore := NewStorageMemory()
	natPinger := func(*traversal.Params) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, mockPaymentEngineFactory, natPinger,
		&MockNatEventTracker{}, "test service id", &mockPublisher{})
	err := manager.Acknowledge(consumerID, "")
	assert.Exactly(t, err, ErrorSessionNotExists)
}

func TestManager_AcknowledgeSession_RejectsBadClient(t *testing.T) {
	sessionStore := NewStorageMemory()
	natPinger := func(*traversal.Params) {}

	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, mockPaymentEngineFactory, natPinger,
		&MockNatEventTracker{}, "test service id", &mockPublisher{})

	pingerParams := &traversal.Params{}
	sessionInstance, err := manager.Create(consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	assert.Nil(t, err)

	err = manager.Acknowledge(identity.FromAddress("some other id"), string(sessionInstance.ID))
	assert.Exactly(t, err, ErrorWrongSessionOwner)
}

func TestManager_AcknowledgeSession_PublishesEvent(t *testing.T) {
	sessionStore := NewStorageMemory()
	natPinger := func(*traversal.Params) {}

	mp := &mockPublisher{}
	manager := NewManager(currentProposal, generateSessionID, sessionStore, mockBalanceTrackerFactory, mockPaymentEngineFactory, natPinger,
		&MockNatEventTracker{}, "test service id", mp)

	pingerParams := &traversal.Params{}
	sessionInstance, err := manager.Create(consumerID, ConsumerInfo{IssuerID: consumerID}, currentProposalID, nil, pingerParams)
	assert.Nil(t, err)

	err = manager.Acknowledge(consumerID, string(sessionInstance.ID))
	assert.Nil(t, err)

	attempts := 0
	for range time.After(time.Millisecond) {
		p := mp.getLast()
		if p.Action == sessionEvent.Acknowledged {
			assert.Equal(t, string(sessionInstance.ID), p.ID)
			return
		}
		attempts++
		if attempts > 50 {
			assert.Fail(t, "no event published")
			return
		}
	}
}
