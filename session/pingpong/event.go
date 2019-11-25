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

// AccountantPromiseTopic represents a topic to which we send accountant promise events.
const AccountantPromiseTopic = "accountant_promise_received"

// AccountantPromiseEventPayload represents the payload that is sent on the AccountantPromiseTopic.
type AccountantPromiseEventPayload struct {
	Promise      crypto.Promise
	AccountantID identity.Identity
	ProviderID   identity.Identity
}
