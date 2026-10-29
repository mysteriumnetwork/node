/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/mysteriumnetwork/node/utils/reftracker"
	"github.com/mysteriumnetwork/payments/crypto"
)

// paymentTestChannel is a p2p channel mock that remembers registered handlers.
type paymentTestChannel struct {
	mockP2PChannel
	mu       sync.Mutex
	handlers map[string]p2p.HandlerFunc
}

func (c *paymentTestChannel) Handle(topic string, handler p2p.HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = handler
}

func (c *paymentTestChannel) handler(topic string) p2p.HandlerFunc {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.handlers[topic]
}

// paymentTestContext is a minimal p2p.Context for calling a handler directly.
type paymentTestContext struct {
	req       *p2p.Message
	peerID    identity.Identity
	publicErr error
}

func (c *paymentTestContext) Request() *p2p.Message            { return c.req }
func (c *paymentTestContext) Error(err error) error            { c.publicErr = err; return nil }
func (c *paymentTestContext) OkWithReply(_ *p2p.Message) error { return nil }
func (c *paymentTestContext) OK() error                        { return nil }
func (c *paymentTestContext) PeerID() identity.Identity        { return c.peerID }

// Example values for a payment message; they only need to parse.
func examplePaymentMessage() *p2p.Message {
	return p2p.ProtoMessage(&pb.ExchangeMessage{
		Promise: &pb.Promise{
			Amount: "100",
			Fee:    "0",
		},
		AgreementID:    "1",
		AgreementTotal: "100",
	})
}

// callPaymentHandler runs the handler in a goroutine and reports whether it
// returned within the wait time, and whether the peer would see an error.
func callPaymentHandler(h p2p.HandlerFunc, peer identity.Identity, wait time.Duration) (returned bool, rejected bool) {
	ctx := &paymentTestContext{req: examplePaymentMessage(), peerID: peer}
	done := make(chan error, 1)
	go func() { done <- h(ctx) }()
	select {
	case err := <-done:
		return true, err != nil || ctx.publicErr != nil
	case <-time.After(wait):
		return false, false
	}
}

func TestSubscribeSessionPayments_RejectsMessagesWithoutActiveSession(t *testing.T) {
	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	ch := &paymentTestChannel{
		mockP2PChannel: mockP2PChannel{tracer: trace.NewTracer("Provider connect")},
		handlers:       map[string]p2p.HandlerFunc{},
	}
	reftracker.Singleton().Put("channel:"+ch.ID(), 10*time.Second, func() { ch.Close() })

	manager := NewSessionManager(
		currentService,
		sessionStore,
		func(_, _ identity.Identity, _ int64, _ common.Address, _ string, _ chan crypto.ExchangeMessage, _ market.Price) (PaymentEngine, error) {
			return &mockBalanceTracker{}, nil
		},
		publisher,
		ch,
		DefaultConfig(),
		&mockPriceValidator{toReturn: true},
	)
	subscribeSessionPayments(manager, ch)
	h := ch.handler(p2p.TopicPaymentMessage)
	require.NotNil(t, h)

	// No session yet: nothing may be queued for it.
	returned, rejected := callPaymentHandler(h, consumerID, time.Second)
	assert.True(t, returned, "handler must not block without a session")
	assert.True(t, rejected, "payment message without an active session")

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
	require.NoError(t, err)
	sess := sessionStore.GetAll()[0]

	// Running session of this consumer: the message is accepted.
	returned, rejected = callPaymentHandler(h, consumerID, time.Second)
	assert.True(t, returned)
	assert.False(t, rejected, "payment message for a running session")

	// Another consumer on this channel has no session here.
	otherConsumer := identity.FromAddress("0x00000000000000000000000000000000000000aa")
	returned, rejected = callPaymentHandler(h, otherConsumer, time.Second)
	assert.True(t, returned)
	assert.True(t, rejected, "payment message from another consumer")

	// Provider closes the session; nobody reads the payment channel any more.
	sess.Close()

	for i := 0; i < 3; i++ {
		returned, rejected = callPaymentHandler(h, consumerID, time.Second)
		assert.True(t, returned, "late payment message %d must not block the handler", i+1)
		assert.True(t, rejected, "late payment message %d after close", i+1)
	}
}

func TestSubscribeSessionPayments_GivesUpWhenQueueStaysFull(t *testing.T) {
	// Example value: keep the test fast instead of waiting the real 20 s.
	defaultTimeout := paymentMessageQueueTimeout
	paymentMessageQueueTimeout = 50 * time.Millisecond
	defer func() { paymentMessageQueueTimeout = defaultTimeout }()

	publisher := mocks.NewEventBus()
	sessionStore := NewSessionPool(publisher)
	ch := &paymentTestChannel{
		mockP2PChannel: mockP2PChannel{tracer: trace.NewTracer("Provider connect")},
		handlers:       map[string]p2p.HandlerFunc{},
	}
	reftracker.Singleton().Put("channel:"+ch.ID(), 10*time.Second, func() { ch.Close() })

	manager := NewSessionManager(
		currentService,
		sessionStore,
		func(_, _ identity.Identity, _ int64, _ common.Address, _ string, _ chan crypto.ExchangeMessage, _ market.Price) (PaymentEngine, error) {
			// This engine never reads the payment channel.
			return &mockBalanceTracker{}, nil
		},
		publisher,
		ch,
		DefaultConfig(),
		&mockPriceValidator{toReturn: true},
	)
	subscribeSessionPayments(manager, ch)
	h := ch.handler(p2p.TopicPaymentMessage)
	require.NotNil(t, h)

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
	require.NoError(t, err)
	defer sessionStore.GetAll()[0].Close()

	// The first message fills the queue (buffer 1).
	returned, rejected := callPaymentHandler(h, consumerID, time.Second)
	assert.True(t, returned)
	assert.False(t, rejected, "first payment message fits in the queue")

	// Nobody reads the queue: the next message must give up instead of blocking.
	returned, rejected = callPaymentHandler(h, consumerID, time.Second)
	assert.True(t, returned, "handler must not block on a full queue")
	assert.True(t, rejected, "payment message that could not be queued in time")
}
