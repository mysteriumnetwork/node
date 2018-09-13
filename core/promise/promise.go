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
	"github.com/mysteriumnetwork/node/money"
)

const endpoint = "promise-create"

// NewPromise creates new Promise object filled by the requested arguments
func NewPromise(consumerID, providerID identity.Identity, amount money.Money) *Promise {
	return &Promise{
		SerialNumber: 1,
		Amount:       amount,
		IssuerID:     consumerID.Address,
		BenefiterID:  providerID.Address,
	}
}

// NewSignedPromise creates a signed promise with a passed signer
func NewSignedPromise(promise *Promise, signer identity.Signer) (*SignedPromise, error) {
	out, err := json.Marshal(promise)
	if err != nil {
		return nil, err
	}
	signature, err := signer.Sign(out)

	return &SignedPromise{
		Promise:         *promise,
		IssuerSignature: Signature(signature.Base64()),
	}, err
}

// Send sends signed promise via the communication channel
func Send(signedPromise *SignedPromise, sender communication.Sender) (*Response, error) {
	responsePtr, err := sender.Request(&Producer{
		SignedPromise: signedPromise,
	})

	response := responsePtr.(*Response)
	if err != nil || !response.Success {
		return nil, errors.New("PromiseDto create failed. " + response.Message)
	}

	return response, nil
}
