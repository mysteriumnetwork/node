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
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/rs/zerolog/log"
)

type orderProvider interface {
	GetPaymentGatewayOrders(id identity.Identity) ([]GatewayOrderResponse, error)
}

type identityProvider interface {
	GetIdentities() []identity.Identity
	IsUnlocked(address string) bool
}

// StatusTracker tracks payment order status.
type StatusTracker struct {
	api              orderProvider
	identityProvider identityProvider
	eventBus         eventbus.Publisher
	orders           map[string]map[string]OrderSummary
	failedSyncs      map[identity.Identity]struct{}

	updateInterval time.Duration
	forceSync      chan identity.Identity
	stopCh         chan struct{}
	once           sync.Once
}

// NewStatusTracker constructs a StatusTracker.
func NewStatusTracker(api orderProvider, identityProvider identityProvider, eventBus eventbus.Publisher, updateInterval time.Duration) *StatusTracker {
	return &StatusTracker{
		api:              api,
		identityProvider: identityProvider,
		eventBus:         eventBus,
		orders:           make(map[string]map[string]OrderSummary),
		failedSyncs:      make(map[identity.Identity]struct{}),
		forceSync:        make(chan identity.Identity),
		updateInterval:   updateInterval,
		stopCh:           make(chan struct{}),
	}
}

// SubscribeAsync subscribes to event to listen for unlock events.
func (t *StatusTracker) SubscribeAsync(bus eventbus.Subscriber) error {
	handleUnlockEvent := func(data identity.AppEventIdentityUnlock) {
		t.UpdateOrdersFor(data.ID)
	}

	// Handle the unlock event so that we load any orders for identities that just launched without
	// waiting for the thread to re-execute.
	return bus.SubscribeAsync(identity.AppTopicIdentityUnlock, handleUnlockEvent)
}

// Track will block and start tracking orders.
func (t *StatusTracker) Track() {
	for {
		select {
		case <-t.stopCh:
			return
		case id := <-t.forceSync:
			t.refreshAndUpdate(id)
		case <-time.After(t.updateInterval):
			for _, id := range t.identityProvider.GetIdentities() {
				if !t.identityProvider.IsUnlocked(id.Address) {
					continue
				}
				if !t.needsRefresh(id) {
					continue
				}

				t.refreshAndUpdate(id)
			}
		}
	}
}

// UpdateOrdersFor sends a notification to the main running thread to
// sync orders for the given identity.
func (t *StatusTracker) UpdateOrdersFor(id identity.Identity) {
	t.forceSync <- id
}

// Stop stops the status tracker
func (t *StatusTracker) Stop() {
	t.once.Do(func() {
		close(t.stopCh)
	})
}

func (t *StatusTracker) needsRefresh(id identity.Identity) bool {
	_, ok := t.failedSyncs[id]
	if ok {
		return true
	}

	orders, ok := t.orders[id.Address]
	if !ok {
		return true
	}

	for _, order := range orders {
		if order.Status.Incomplete() {
			return true
		}
	}

	return false
}

func (t *StatusTracker) refreshAndUpdate(id identity.Identity) {
	// If we fail to sync or only sync partialy we must force a repeat
	t.failedSyncs[id] = struct{}{}
	defer delete(t.failedSyncs, id)

	newOrders, err := t.refresh(id)
	if err != nil {
		log.Err(err).Str("identity", id.Address).Msg("Could not update orders")
		return
	}

	if len(newOrders) == 0 {
		if _, ok := t.orders[id.Address]; !ok {
			t.orders[id.Address] = map[string]OrderSummary{}
		}
		return
	}

	t.compareAndUpdate(id, newOrders)
}

func (t *StatusTracker) compareAndUpdate(id identity.Identity, newOrders map[string]OrderSummary) {
	old, ok := t.orders[id.Address]
	if !ok || len(old) == 0 {
		t.orders[id.Address] = newOrders
		return
	}

	updated := make(map[string]OrderSummary)
	for _, no := range newOrders {
		old, ok := old[no.ID]
		if !ok {
			updated[no.ID] = no
			if no.Status.Incomplete() {
				continue
			}

			// If the entry is new but already completed, send an update about it
			t.eventBus.Publish(AppTopicOrderUpdated, AppEventOrderUpdated{no})
		}

		newEntry, changed := applyChanges(old, no)
		if changed {
			t.eventBus.Publish(AppTopicOrderUpdated, AppEventOrderUpdated{newEntry})
		}
		updated[no.ID] = newEntry
	}

	t.orders[id.Address] = updated
}

func (t *StatusTracker) refresh(id identity.Identity) (map[string]OrderSummary, error) {
	result := make(map[string]OrderSummary)
	gwOrders, err := t.api.GetPaymentGatewayOrders(id)
	if err != nil {
		return nil, err
	}

	for _, o := range gwOrders {
		result[o.ID] = OrderSummary{
			ID:              o.ID,
			IdentityAddress: o.Identity,
			Status:          o.Status,
			PayAmount:       o.PayAmount,
			PayCurrency:     o.PayCurrency,
		}
	}

	return result, nil
}

// applyChanges applies changes to the OrderSummary from an OrderResponse. Returns true if changed.
func applyChanges(order OrderSummary, newOrder OrderSummary) (OrderSummary, bool) {
	changed := false
	if order.Status != newOrder.Status {
		order.Status = newOrder.Status
		changed = true
	}
	if order.PayAmount != newOrder.PayAmount {
		order.PayAmount = newOrder.PayAmount
		changed = true
	}
	if order.PayCurrency != newOrder.PayCurrency {
		order.PayCurrency = newOrder.PayCurrency
		changed = true
	}

	return order, changed
}

// OrderSummary is a subset of an OrderResponse stored by the StatusTracker.
type OrderSummary struct {
	ID              string
	IdentityAddress string
	Status          CompletionProvider
	PayAmount       string
	PayCurrency     string
}

// CompletionProvider is a temporary interface to make
// any order work with the tracker.
// TODO: Remove after legacy payments are removed.
type CompletionProvider interface {
	Incomplete() bool
	Status() string
	Paid() bool
}

func (o OrderSummary) String() string {
	return fmt.Sprintf("ID: %v, IdentityAddress: %v, Status: %v, PayAmount: %v, PayCurrency: %v", o.ID, o.IdentityAddress, o.Status, o.PayAmount, o.PayCurrency)
}
