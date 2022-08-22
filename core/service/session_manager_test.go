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
	"errors"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/mysteriumnetwork/node/utils/reftracker"
	"github.com/mysteriumnetwork/payments/crypto"
)

var (
	currentProposalID = 68
	currentProposal   = market.NewProposal("0x1", "mockservice", market.NewProposalOpts{})
	currentService    = NewInstance(
		identity.FromAddress(currentProposal.ProviderID),
		currentProposal.ServiceType,
		struct{}{},
		currentProposal,
		servicestate.Running,
		&mockService{},
		policy.NewRepository(),
		&mockDiscovery{},
	)
	consumerID = identity.FromAddress("deadbeef")
	hermesID   = common.HexToAddress("0x1")
)

type mockBalanceTracker struct {
	paymentError      error
	firstPaymentError error
}

func (m mockBalanceTracker) Start() error {
	return m.paymentError
}

func (m mockBalanceTracker) Stop() {
}

func (m mockBalanceTracker) WaitFirstInvoice(time.Duration) error {
	return m.firstPaymentError
}

type mockP2PChannel struct {
	tracer *trace.Tracer
}

func (m *mockP2PChannel) Send(_ context.Context, _ string, _ *p2p.Message) (*p2p.Message, error) {
	return nil, nil
}

func (m *mockP2PChannel) Handle(topic string, handler p2p.HandlerFunc) {
}

func (m *mockP2PChannel) Tracer() *trace.Tracer {
	return m.tracer
}

func (m *mockP2PChannel) ServiceConn() *net.UDPConn { return nil }

func (m *mockP2PChannel) Conn() *net.UDPConn { return nil }

func (m *mockP2PChannel) Close() error { return nil }

func (m *mockP2PChannel) ID() string { return fmt.Sprintf("%p", m) }

func TestManager_Start_StoresSession(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, true)

	_, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:       consumerID.Address,
			HermesID: hermesID.String(),
			Pricing: &pb.Pricing{
				PerGib:  big.NewInt(1).Bytes(),
				PerHour: big.NewInt(1).Bytes(),
			},
		},
		ProposalID: int64(currentProposalID),
	})
	assert.NoError(t, err)

	session := sessionStore.GetAll()[0]
	assert.Equal(t, consumerID, session.ConsumerID)

	assert.Eventually(t, func() bool {
		history := publisher.GetEventHistory()
		if len(history) != 6 {
			return false
		}

		assert.Equal(t, sessionEvent.AppTopicSession, history[0].Topic)
		startEvent := history[0].Event.(sessionEvent.AppEventSession)
		assert.Equal(t, sessionEvent.CreatedStatus, startEvent.Status)
		assert.Equal(t, consumerID, startEvent.Session.ConsumerID)
		assert.Equal(t, hermesID, startEvent.Session.HermesID)
		assert.Equal(t, currentProposal, startEvent.Session.Proposal)

		assert.Equal(t, trace.AppTopicTraceEvent, history[1].Topic)
		traceEvent1 := history[1].Event.(trace.Event)
		assert.Equal(t, "Provider connect", traceEvent1.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[2].Topic)
		traceEvent2 := history[2].Event.(trace.Event)
		assert.Equal(t, "Provider session create", traceEvent2.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[3].Topic)
		traceEvent3 := history[3].Event.(trace.Event)
		assert.Equal(t, "Provider session create (start)", traceEvent3.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[4].Topic)
		traceEvent4 := history[4].Event.(trace.Event)
		assert.Equal(t, "Provider session create (payment)", traceEvent4.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[5].Topic)
		traceEvent5 := history[5].Event.(trace.Event)
		assert.Equal(t, "Provider session create (configure)", traceEvent5.Key)

		return true
	}, 2*time.Second, 10*time.Millisecond)
}

func TestManager_Start_DisconnectsOnPaymentError(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{
		firstPaymentError: errors.New("sorry, your money ended"),
	}, true)

	_, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:       consumerID.Address,
			HermesID: hermesID.String(),
			Pricing: &pb.Pricing{
				PerGib:  big.NewInt(1).Bytes(),
				PerHour: big.NewInt(1).Bytes(),
			},
		},
		ProposalID: int64(currentProposalID),
	})
	assert.EqualError(t, err, "first invoice was not paid: sorry, your money ended")
	assert.Eventually(t, func() bool {
		history := publisher.GetEventHistory()
		if len(history) != 6 {
			return false
		}

		assert.Equal(t, sessionEvent.AppTopicSession, history[0].Topic)
		startEvent := history[0].Event.(sessionEvent.AppEventSession)
		assert.Equal(t, sessionEvent.CreatedStatus, startEvent.Status)
		assert.Equal(t, consumerID, startEvent.Session.ConsumerID)
		assert.Equal(t, hermesID, startEvent.Session.HermesID)
		assert.Equal(t, currentProposal, startEvent.Session.Proposal)

		assert.Equal(t, trace.AppTopicTraceEvent, history[1].Topic)
		traceEvent1 := history[1].Event.(trace.Event)
		assert.Equal(t, "Provider connect", traceEvent1.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[2].Topic)
		traceEvent2 := history[2].Event.(trace.Event)
		assert.Equal(t, "Provider session create", traceEvent2.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[3].Topic)
		traceEvent3 := history[3].Event.(trace.Event)
		assert.Equal(t, "Provider session create (start)", traceEvent3.Key)

		assert.Equal(t, trace.AppTopicTraceEvent, history[4].Topic)
		traceEvent4 := history[4].Event.(trace.Event)
		assert.Equal(t, "Provider session create (payment)", traceEvent4.Key)

		assert.Equal(t, sessionEvent.AppTopicSession, history[5].Topic)
		closeEvent := history[5].Event.(sessionEvent.AppEventSession)
		assert.Equal(t, sessionEvent.RemovedStatus, closeEvent.Status)
		assert.Equal(t, consumerID, closeEvent.Session.ConsumerID)
		assert.Equal(t, hermesID, closeEvent.Session.HermesID)
		assert.Equal(t, currentProposal, closeEvent.Session.Proposal)

		return true
	}, 2*time.Second, 10*time.Millisecond)
}

func TestManager_Start_Second_Session_Destroy_Stale_Session(t *testing.T) {
	sessionRequest := &pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:       consumerID.Address,
			HermesID: hermesID.String(),
			Pricing: &pb.Pricing{
				PerGib:  big.NewInt(1).Bytes(),
				PerHour: big.NewInt(1).Bytes(),
			},
		},
		ProposalID: int64(currentProposalID),
	}

	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, true)

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
	}, 2*time.Second, 10*time.Millisecond, "Waiting for session destroy")
}

func TestManager_AcknowledgeSession_RejectsUnknown(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, true)

	err := manager.Acknowledge(consumerID, "")
	assert.Exactly(t, err, ErrorSessionNotExists)
}

func TestManager_AcknowledgeSession_RejectsBadClient(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(mocks.NewEventBus())
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, true)

	session, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:       consumerID.Address,
			HermesID: hermesID.String(),
			Pricing: &pb.Pricing{
				PerGib:  big.NewInt(1).Bytes(),
				PerHour: big.NewInt(1).Bytes(),
			},
		},
		ProposalID: int64(currentProposalID),
	})
	assert.Nil(t, err)

	err = manager.Acknowledge(identity.FromAddress("some other id"), string(session.ID))
	assert.Exactly(t, ErrorWrongSessionOwner, err)
}

func TestManager_AcknowledgeSession_PublishesEvent(t *testing.T) {
	publisher := mocks.NewEventBus()

	sessionStore := NewSessionPool(publisher)
	session, _ := NewSession(
		currentService,
		&pb.SessionRequest{Consumer: &pb.ConsumerInfo{Id: consumerID.Address}},
		trace.NewTracer(""),
	)
	sessionStore.Add(session)

	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, true)

	err := manager.Acknowledge(consumerID, string(session.ID))
	assert.Nil(t, err)
	assert.Eventually(t, func() bool {
		// Check that state event with StateIPNotChanged status was called.
		history := publisher.GetEventHistory()
		for _, v := range history {
			if v.Topic == sessionEvent.AppTopicSession && v.Event.(sessionEvent.AppEventSession).Status == sessionEvent.AcknowledgedStatus {
				return true
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond)
}

func newManager(service *Instance, sessions *SessionPool, publisher publisher, paymentEngine PaymentEngine, isPriceValid bool) *SessionManager {
	ch := &mockP2PChannel{tracer: trace.NewTracer("Provider connect")}
	m := NewSessionManager(
		service,
		sessions,
		func(_, _ identity.Identity, _ int64, _ common.Address, _ string, _ chan crypto.ExchangeMessage, price market.Price) (PaymentEngine, error) {
			return paymentEngine, nil
		},
		publisher,
		ch,
		DefaultConfig(),
		&mockPriceValidator{
			toReturn: isPriceValid,
		},
	)
	reftracker.Singleton().Put("channel:"+ch.ID(), 10*time.Second, func() { ch.Close() })
	return m
}

func TestManager_Start_RejectsInvalidPricing(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	manager := newManager(currentService, sessionStore, publisher, &mockBalanceTracker{}, false)

	_, err := manager.Start(&pb.SessionRequest{
		Consumer: &pb.ConsumerInfo{
			Id:       consumerID.Address,
			HermesID: hermesID.String(),
			Pricing: &pb.Pricing{
				PerGib:  big.NewInt(1).Bytes(),
				PerHour: big.NewInt(1).Bytes(),
			},
		},
		ProposalID: int64(currentProposalID),
	})
	assert.Error(t, err)
	assert.Equal(t, "consumer asking for invalid price", err.Error())
}

type mockPriceValidator struct {
	toReturn bool
}

func (mpv *mockPriceValidator) IsPriceValid(in market.Price, nodeType, country string) bool {
	return mpv.toReturn
}
