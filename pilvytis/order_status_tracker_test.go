/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package pilvytis

import (
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/stretchr/testify/assert"
)

func TestStatusTracker(t *testing.T) {
	// given
	mockAPI := &mockAPI{orders: []OrderResponse{{
		ID:     1,
		Status: OrderStatusNew,
	}}}
	bus := mocks.NewEventBus()
	tracker := NewStatusTracker(mockAPI, mockIdentityProvider{}, bus, time.Millisecond*2)
	defer tracker.Pause()

	// when
	tracker.Track()
	mockAPI.returns([]OrderResponse{{
		ID:     1,
		Status: OrderStatusConfirming,
	}})
	// then
	assert.Eventually(t, EventPublished(bus, 1, OrderStatusConfirming), time.Millisecond*500, 5*time.Millisecond)

	// when
	tracker.Pause()
	mockAPI.returns([]OrderResponse{{
		ID:     1,
		Status: OrderStatusPaid,
	}})
	// then
	assert.Never(t, EventPublished(bus, 1, OrderStatusPaid), time.Millisecond*500, 5*time.Millisecond)

	// when
	tracker.Track()
	// then
	assert.Eventually(t, EventPublished(bus, 1, OrderStatusPaid), time.Millisecond*500, 5*time.Millisecond)

	// when
	mockAPI.returns([]OrderResponse{{
		ID:     1,
		Status: OrderStatusPaid,
	}, {
		ID:     2,
		Status: OrderStatusPaid,
	}})
	tracker.Track()
	// then
	assert.Eventually(t, EventPublished(bus, 2, OrderStatusPaid), time.Millisecond*500, 5*time.Millisecond)
}

type mockAPI struct {
	orders []OrderResponse
	sync.RWMutex
}

func (m *mockAPI) returns(orders []OrderResponse) {
	m.Lock()
	defer m.Unlock()
	m.orders = orders
}

func (m *mockAPI) GetPaymentOrders(_ identity.Identity) ([]OrderResponse, error) {
	m.RLock()
	defer m.RUnlock()
	return m.orders, nil
}

type mockIdentityProvider struct {
}

func (m mockIdentityProvider) GetIdentities() []identity.Identity {
	return []identity.Identity{identity.FromAddress("0xbeef")}
}

func EventPublished(bus *mocks.EventBus, orderID uint64, status OrderStatus) func() bool {
	return func() bool {
		pop := bus.Pop()
		if pop == nil {
			return false
		}
		evt, ok := pop.(AppEventOrderUpdated)
		return ok && evt.ID == orderID && evt.Status == status
	}
}
