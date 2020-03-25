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
	Previous uint64
	Current  uint64
}
