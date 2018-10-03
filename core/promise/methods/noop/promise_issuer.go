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

package noop

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
)

const issuerLogPrefix = "[promise-issuer] "

// PromiseIssuer issues promises in such way, that no actual money is added to promise
type PromiseIssuer struct {
	IssuerID identity.Identity
	Dialog   communication.Dialog
	Signer   identity.Signer
	Amount   money.Money

	// these are populated by Start at runtime
	proposal dto.ServiceProposal
}

// Start issuing promises for given service proposal
func (issuer *PromiseIssuer) Start(proposal dto.ServiceProposal) error {
	issuer.proposal = proposal

	return issuer.subscribePromiseBalance()
}

// Stop stops issuing promises
func (issuer *PromiseIssuer) Stop() error {
	// TODO Should unregister consumers(subscriptions) here
	return nil
}

func (issuer *PromiseIssuer) subscribePromiseBalance() error {
	return issuer.Dialog.Receive(
		&promise.BalanceMessageConsumer{issuer.processBalanceMessage},
	)
}

func (issuer *PromiseIssuer) processBalanceMessage(message promise.BalanceMessage) error {
	if !message.Accepted {
		log.Warn(issuerLogPrefix, fmt.Sprintf("Promise balance rejected: %s", message.Balance.String()))
	}

	log.Info(issuerLogPrefix, fmt.Sprintf("Promise balance notified: %s", message.Balance.String()))
	return nil
}
