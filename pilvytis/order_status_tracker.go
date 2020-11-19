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
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/rs/zerolog/log"
)

// OrderSummary is a subset of an OrderResponse stored by the StatusTracker.
type OrderSummary struct {
	ID              uint64
	IdentityAddress string
	Status          OrderStatus
}

type orderProvider interface {
	GetPaymentOrders(id identity.Identity) ([]OrderResponse, error)
}

// StatusTracker tracks payment order status.
type StatusTracker struct {
	api              orderProvider
	identityProvider identityProvider
	eventBus         eventbus.Publisher
	orders           []OrderSummary

	updateInterval time.Duration
	stopCh         chan struct{}
	tracking       bool
	lock           sync.Mutex
}

// NewStatusTracker constructs a StatusTracker.
func NewStatusTracker(api orderProvider, identityProvider identityProvider, eventBus eventbus.Publisher, updateInterval time.Duration) *StatusTracker {
	return &StatusTracker{
		api:              api,
		identityProvider: identityProvider,
		eventBus:         eventBus,
		orders:           []OrderSummary{},
		updateInterval:   updateInterval,
		stopCh:           make(chan struct{}),
	}
}

func (t *StatusTracker) getOrCreate(id uint64, identityAddress string) *OrderSummary {
	for i := range t.orders {
		if t.orders[i].ID == id {
			return &t.orders[i]
		}
	}
	newEntry := OrderSummary{ID: id, IdentityAddress: identityAddress}
	t.orders = append(t.orders, newEntry)
	return &newEntry
}

// Track order status updates.
func (t *StatusTracker) Track() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.tracking {
		return
	}
	t.tracking = true
	go func() {
		log.Info().Msg("Tracking order statuses...")
		updateTicker := time.NewTicker(t.updateInterval)
		defer updateTicker.Stop()
		for {
			select {
			case <-t.stopCh:
				return
			case <-updateTicker.C:
				err := t.update()
				if err != nil {
					log.Err(err).Msg("Could not update order statuses")
				}
			}
		}
	}()
}

// Pause status tracking.
func (t *StatusTracker) Pause() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.tracking {
		return
	}
	log.Info().Msg("Pausing order status tracking")
	t.stopCh <- struct{}{}
	t.tracking = false
}

func (t *StatusTracker) update() error {
	log.Info().Msg("Updating order statuses")
	keepTracking := false
	for _, id := range t.identityProvider.GetIdentities() {
		newOrders, err := t.api.GetPaymentOrders(id)
		if err != nil {
			log.Warn().Str("identity", id.Address).Msg("Could not update orders")
			keepTracking = true
			continue
		}
		for _, newOrder := range newOrders {
			order := t.getOrCreate(newOrder.ID, newOrder.Identity)
			if order.Status != newOrder.Status {
				order.Status = newOrder.Status
				t.eventBus.Publish(AppTopicOrderStatusChanged, AppEventOrderStatusChanged{*order})
			}
			if newOrder.Status.Incomplete() {
				keepTracking = true
			}
		}
	}
	if !keepTracking {
		go t.Pause()
	}
	return nil
}
