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
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// PromiseIssuer issues promises in such way, that no actual money is added to promise
type PromiseIssuer struct {
	issuerID identity.Identity
	dialog   communication.Dialog
	signer   identity.Signer

	// these are populated by Start at runtime
	proposal market.ServiceProposal
}

// NewPromiseIssuer creates instance of the promise issuer
func NewPromiseIssuer(issuerID identity.Identity, dialog communication.Dialog, signer identity.Signer) *PromiseIssuer {
	return &PromiseIssuer{issuerID: issuerID, dialog: dialog, signer: signer}
}

// Start issuing promises for given service proposal
func (issuer *PromiseIssuer) Start(proposal market.ServiceProposal) error {
	issuer.proposal = proposal

	if err := issuer.sendNewPromise(); err != nil {
		return err
	}

	return issuer.subscribePromiseBalance()
}

// Stop stops issuing promises
func (issuer *PromiseIssuer) Stop() error {
	// TODO Should unregister consumers(subscriptions) here
	return nil
}

func (issuer *PromiseIssuer) sendNewPromise() error {
	unsignedPromise := promise.NewPromise(
		issuer.issuerID,
		identity.FromAddress(issuer.proposal.ProviderID),
		issuer.proposal.PaymentMethod.GetPrice())

	signedPromise, err := unsignedPromise.SignByIssuer(issuer.signer)
	if err != nil {
		return err
	}

	return signedPromise.Send(issuer.dialog)
}

func (issuer *PromiseIssuer) subscribePromiseBalance() error {
	return issuer.dialog.Receive(
		&promise.BalanceMessageConsumer{Callback: issuer.processBalanceMessage},
	)
}

func (issuer *PromiseIssuer) processBalanceMessage(message promise.BalanceMessage) error {
	if !message.Accepted {
		log.Warnf("promise balance rejected: %s", message.Balance.String())
	}

	log.Infof("promise balance notified: %s", message.Balance.String())
	return nil
}
