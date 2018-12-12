/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

// Package promise allows client to issue proposal, that will be sent to the provider.
// It's provider responsibility to store and process promises.
package promise

import (
	"encoding/json"
	"errors"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

const endpoint = "promise-create"

var (
	errLowAmount          = errors.New("promise amount less than the service proposal price")
	errLowBalance         = errors.New("issuer balance less than the promise amount")
	errBadSignature       = errors.New("invalid Signature for the provided identity")
	errUnknownBenefiter   = errors.New("unknown promise benefiter received")
	errUnsupportedRequest = errors.New("unsupported request")
)

// NewPromise creates new Promise object filled by the requested arguments
func NewPromise(issuerID, benefiterID identity.Identity, amount money.Money) *Promise {
	return &Promise{
		SerialNumber: 1,
		Amount:       amount,
		IssuerID:     issuerID.Address,
		BenefiterID:  benefiterID.Address,
	}
}

// SignByIssuer creates a signed promise with a passed issuerSigner
func (p *Promise) SignByIssuer(issuerSigner identity.Signer) (*SignedPromise, error) {
	out, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	signature, err := issuerSigner.Sign(out)

	return &SignedPromise{
		Promise:         *p,
		IssuerSignature: Signature(signature.Base64()),
	}, err
}

// Send sends signed promise via the communication channel
func (sp *SignedPromise) Send(sender communication.Sender) error {
	responsePtr, err := sender.Request(&Producer{SignedPromise: sp})

	response := responsePtr.(*Response)
	if err != nil || !response.Success {
		return errors.New("Promise issuing failed: " + response.Message)
	}

	return nil
}

// Validate check signed promise to be valid. It checks signature, benefiter address.
// Also it compares the promised amount to be enough for the proposal.
// And finally it checks that issuer have enough balance to issue the promice.
func (sp *SignedPromise) Validate(proposal market.ServiceProposal, balance identity.Balance) error {
	receivedPromise, err := json.Marshal(sp.Promise)
	if err != nil {
		return err
	}

	signature := identity.SignatureBase64(string(sp.IssuerSignature))
	issuer := identity.FromAddress(sp.Promise.IssuerID)
	verifier := identity.NewVerifierIdentity(issuer)
	if !verifier.Verify(receivedPromise, signature) {
		return errBadSignature
	}

	benefiter := identity.FromAddress(sp.Promise.BenefiterID)
	if benefiter.Address != proposal.ProviderID {
		return errUnknownBenefiter
	}

	price := proposal.PaymentMethod.GetPrice()
	promisedValue := sp.Promise.Amount
	if promisedValue.Amount < price.Amount {
		return errLowAmount
	}

	issuerBalance, err := balance(issuer)
	if err != nil {
		return err
	}
	if issuerBalance < promisedValue.Amount {
		return errLowBalance
	}

	return nil
}
