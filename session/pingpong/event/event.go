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
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

const (
	// AppTopicHermesPromise represents a topic to which we send hermes promise events.
	AppTopicHermesPromise = "hermes_promise_received"
	// AppTopicBalanceChanged represents the balance change topic
	AppTopicBalanceChanged = "balance_change"
	// AppTopicEarningsChanged represents the earnings change topic
	AppTopicEarningsChanged = "earnings_change"
	// AppTopicInvoicePaid is a topic for publish events exchange message send to provider as a consumer.
	AppTopicInvoicePaid = "invoice_paid"
	// AppTopicSettlementRequest forces the settlement of promises for given provider/hermes.
	AppTopicSettlementRequest = "settlement_request"
	// AppTopicSettlementComplete topic for events related to completed settlement.
	AppTopicSettlementComplete = "provider_settlement_complete"
	// AppTopicWithdrawalRequested topic for succesfull withdrawal requests.
	AppTopicWithdrawalRequested = "provider_withdrawal_requested"
)

// AppEventSettlementRequest represents the payload that is sent on the AppTopicSettlementRequest topic.
type AppEventSettlementRequest struct {
	HermesID   common.Address
	ProviderID identity.Identity
	ChainID    int64
}

// AppEventHermesPromise represents the payload that is sent on the AppTopicHermesPromise.
type AppEventHermesPromise struct {
	Promise    crypto.Promise
	HermesID   common.Address
	ProviderID identity.Identity
}

// AppEventBalanceChanged represents a balance change event
type AppEventBalanceChanged struct {
	Identity identity.Identity
	Previous *big.Int
	Current  *big.Int
}

// AppEventEarningsChanged represents a balance change event
type AppEventEarningsChanged struct {
	Identity identity.Identity
	Previous EarningsDetailed
	Current  EarningsDetailed
}

// EarningsDetailed returns total and split per hermes earnings
type EarningsDetailed struct {
	Total     Earnings
	PerHermes map[common.Address]Earnings
}

// Earnings represents current identity earnings
type Earnings struct {
	LifetimeBalance  *big.Int
	UnsettledBalance *big.Int
}

// AppEventInvoicePaid is an update on paid invoices during current session
type AppEventInvoicePaid struct {
	UUID       string
	ConsumerID identity.Identity
	SessionID  string
	Invoice    crypto.Invoice
}

// AppTopicGrandTotalChanged represents a topic to which we send grand total change messages.
const AppTopicGrandTotalChanged = "consumer_grand_total_change"

// AppEventGrandTotalChanged represents the grand total changed event.
type AppEventGrandTotalChanged struct {
	Current    *big.Int
	ChainID    int64
	HermesID   common.Address
	ConsumerID identity.Identity
}

// AppEventSettlementComplete represent a completed settlement.
type AppEventSettlementComplete struct {
	ProviderID identity.Identity
	HermesID   common.Address
	ChainID    int64
}

// AppEventWithdrawalRequested represents a request for withdrawal.
type AppEventWithdrawalRequested struct {
	ProviderID         identity.Identity
	HermesID           common.Address
	FromChain, ToChain int64
}
