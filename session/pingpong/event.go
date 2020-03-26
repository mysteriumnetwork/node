/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
)

// AppTopicAccountantPromise represents a topic to which we send accountant promise events.
const AppTopicAccountantPromise = "accountant_promise_received"

// AppEventAccountantPromise represents the payload that is sent on the AppTopicAccountantPromise.
type AppEventAccountantPromise struct {
	Promise      crypto.Promise
	AccountantID identity.Identity
	ProviderID   identity.Identity
}

// AppTopicExchangeMessage represents a topic where exchange messages are sent.
const AppTopicExchangeMessage = "exchange_message_topic"

// AppEventExchangeMessage represents the messages that are sent on the AppTopicExchangeMessage.
type AppEventExchangeMessage struct {
	Identity       identity.Identity
	AmountPromised uint64
}

// AppTopicBalanceChanged represents the balance change topic
const AppTopicBalanceChanged = "balance_change"

// AppEventBalanceChanged represents a balance change event
type AppEventBalanceChanged struct {
	Identity identity.Identity
	Previous uint64
	Current  uint64
}

// AppTopicEarningsChanged represents the earnings change topic
const AppTopicEarningsChanged = "earnings_change"

// AppEventEarningsChanged represents a balance change event
type AppEventEarningsChanged struct {
	Identity identity.Identity
	Previous SettlementState
	Current  SettlementState
}

// SettlementState represents current settling state with values of identity earnings
type SettlementState struct {
	Channel     client.ProviderChannel
	LastPromise crypto.Promise

	settleInProgress bool
	registered       bool
}

// LifetimeBalance returns earnings of all history.
func (ss SettlementState) LifetimeBalance() uint64 {
	return ss.LastPromise.Amount
}

// UnsettledBalance returns current unsettled earnings.
func (ss SettlementState) UnsettledBalance() uint64 {
	settled := uint64(0)
	if ss.Channel.Settled != nil {
		settled = ss.Channel.Settled.Uint64()
	}

	return safeSub(ss.LastPromise.Amount, settled)
}

func (ss SettlementState) availableBalance() uint64 {
	balance := uint64(0)
	if ss.Channel.Balance != nil {
		balance = ss.Channel.Balance.Uint64()
	}

	settled := uint64(0)
	if ss.Channel.Settled != nil {
		settled = ss.Channel.Settled.Uint64()
	}

	return balance + settled
}

func (ss SettlementState) balance() uint64 {
	return safeSub(ss.availableBalance(), ss.LastPromise.Amount)
}

func (ss SettlementState) needsSettling(threshold float64) bool {
	if !ss.registered {
		return false
	}

	if ss.settleInProgress {
		return false
	}

	if float64(ss.balance()) <= 0 {
		return true
	}

	if float64(ss.balance()) <= threshold*float64(ss.availableBalance()) {
		return true
	}

	return false
}

// AppTopicGrandTotalRecovered represents a topic to which we send grand total recovered messages.
const AppTopicGrandTotalRecovered = "consumer_grand_total_recovered"

// GrandTotalRecovered represents the AppTopicGrandTotalRecovered payload.
type GrandTotalRecovered struct {
	Identity identity.Identity
}
