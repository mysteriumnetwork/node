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

package event

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

const (
	// AppTopicAccountantPromise represents a topic to which we send accountant promise events.
	AppTopicAccountantPromise = "accountant_promise_received"
	// AppTopicBalanceChanged represents the balance change topic
	AppTopicBalanceChanged = "balance_change"
	// AppTopicEarningsChanged represents the earnings change topic
	AppTopicEarningsChanged = "earnings_change"
	// AppTopicInvoicePaid is a topic for publish events exchange message send to provider as a consumer.
	AppTopicInvoicePaid = "invoice_paid"
)

// AppEventAccountantPromise represents the payload that is sent on the AppTopicAccountantPromise.
type AppEventAccountantPromise struct {
	Promise      crypto.Promise
	AccountantID common.Address
	ProviderID   identity.Identity
}

// AppEventBalanceChanged represents a balance change event
type AppEventBalanceChanged struct {
	Identity identity.Identity
	Previous uint64
	Current  uint64
}

// AppEventEarningsChanged represents a balance change event
type AppEventEarningsChanged struct {
	Identity identity.Identity
	Previous Earnings
	Current  Earnings
}

// Earnings represents current identity earnings
type Earnings struct {
	LifetimeBalance  uint64
	UnsettledBalance uint64
}

// AppEventInvoicePaid is an update on paid invoices during current session
type AppEventInvoicePaid struct {
	ConsumerID identity.Identity
	SessionID  string
	Invoice    crypto.Invoice
}

// AppTopicGrandTotalChanged represents a topic to which we send grand total change messages.
const AppTopicGrandTotalChanged = "consumer_grand_total_change"

// AppEventGrandTotalChanged represents the grand total changed event.
type AppEventGrandTotalChanged struct {
	Current      uint64
	AccountantID common.Address
	ConsumerID   identity.Identity
}
