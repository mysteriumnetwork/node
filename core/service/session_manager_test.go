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

package service

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

var (
	currentProposalID = 68
	currentProposal   = market.ServiceProposal{
		ServiceType: "mockservice",
		ID:          currentProposalID,
	}
	currentService = NewInstance(
		identity.FromAddress(currentProposal.ProviderID),
		currentProposal.ServiceType,
		struct{}{},
		currentProposal,
		servicestate.Running,
		&mockService{},
		policy.NewRepository(),
		&mockDiscovery{},
	)
	consumerID   = identity.FromAddress("deadbeef")
	accountantID = common.HexToAddress("0x1")
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

type mockP2PChannel struct{}

func (m *mockP2PChannel) Send(_ context.Context, _ string, _ *p2p.Message) (*p2p.Message, error) {
	return nil, nil
}

func (m *mockP2PChannel) Handle(topic string, handler p2p.HandlerFunc) {
}

func (m *mockP2PChannel) ServiceConn() *net.UDPConn { return nil }

func (m *mockP2PChannel) Conn() *net.UDPConn { return nil }

func (m *mockP2PChannel) Close() error { return nil }

func TestManager_Start_StoresSession(t *testing.T) {
	sessionStore := NewSessionPool(mocks.NewEventBus())

	manager := newManager(currentService, sessionStore)

	_, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:           consumerID.Address,
			AccountantID: accountantID.String(),
		},
		ProposalID: int64(currentProposalID),
	})
	assert.NoError(t, err)

	session := sessionStore.GetAll()[0]
	assert.Equal(t, consumerID, session.ConsumerID)
}

func TestManager_Start_Second_Session_Destroy_Stale_Session(t *testing.T) {
	sessionRequest := &pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:           consumerID.Address,
			AccountantID: accountantID.String(),
		},
		ProposalID: int64(currentProposalID),
	}

	sessionStore := NewSessionPool(mocks.NewEventBus())
	manager := newManager(currentService, sessionStore)

	_, err := manager.Start(sessionRequest)
	assert.NoError(t, err)

	sessionOld := sessionStore.GetAll()[0]
	assert.Equal(t, consumerID, sessionOld.ConsumerID)

	_, err = manager.Start(sessionRequest)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Eventuallyf(t, func() bool {
		_, found := sessionStore.Find(sessionOld.ID)
		return !found
	}, time.Second, 10*time.Millisecond, "Waiting for session destroy")
}

func TestManager_Start_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewSessionPool(mocks.NewEventBus())

	manager := newManager(currentService, sessionStore)

	_, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:           consumerID.Address,
			AccountantID: accountantID.String(),
		},
		ProposalID: int64(69),
	})
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Len(t, sessionStore.GetAll(), 0)
}

type MockNatEventTracker struct {
}

func (mnet *MockNatEventTracker) LastEvent() *event.Event {
	return &event.Event{}
}

func TestManager_AcknowledgeSession_RejectsUnknown(t *testing.T) {
	sessionStore := NewSessionPool(mocks.NewEventBus())

	manager := newManager(currentService, sessionStore)
	err := manager.Acknowledge(consumerID, "")
	assert.Exactly(t, err, ErrorSessionNotExists)
}

func TestManager_AcknowledgeSession_RejectsBadClient(t *testing.T) {
	sessionStore := NewSessionPool(mocks.NewEventBus())

	manager := newManager(currentService, sessionStore)

	session, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:           consumerID.Address,
			AccountantID: accountantID.String(),
		},
		ProposalID: int64(currentProposalID),
	})
	assert.Nil(t, err)

	err = manager.Acknowledge(identity.FromAddress("some other id"), string(session.ID))
	assert.Exactly(t, ErrorWrongSessionOwner, err)
}

func TestManager_AcknowledgeSession_PublishesEvent(t *testing.T) {
	sessionStore := NewSessionPool(mocks.NewEventBus())
	session := Session{ID: "1", ConsumerID: consumerID}
	sessionStore.Add(session)

	mp := mocks.NewEventBus()
	manager := newManager(currentService, sessionStore)
	manager.publisher = mp

	err := manager.Acknowledge(consumerID, string(session.ID))
	assert.Nil(t, err)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.AcknowledgedStatus), 2*time.Second, 10*time.Millisecond)
}

func newManager(service *Instance, sessions *SessionPool) *SessionManager {
	return NewSessionManager(service, sessions, mockPaymentEngineFactory, &MockNatEventTracker{}, mocks.NewEventBus(), &mockP2PChannel{}, DefaultConfig())
}
