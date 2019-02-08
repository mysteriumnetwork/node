/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package issuers

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/promise"
	payments_identity "github.com/mysteriumnetwork/payments/identity"
	"github.com/mysteriumnetwork/payments/promises"
)

// LocalIssuer issues signed promise by using identity.Signer (usually based on local keystore)
type LocalIssuer struct {
	paymentsSigner payments_identity.Signer
}

// NewLocalIssuer creates local issuer based on provided identity signer
func NewLocalIssuer(signer identity.Signer) *LocalIssuer {
	return &LocalIssuer{
		paymentsSigner: paymentsSignerAdapter{
			identitySigner: signer,
		},
	}
}

// Issue issues the given promise
func (li LocalIssuer) Issue(promise promises.Promise) (promises.IssuedPromise, error) {
	signed, err := promises.SignByPayer(&promise, li.paymentsSigner)
	if err != nil {
		// TODO this looks ugly - align interface or discard pointers to structs?
		return promises.IssuedPromise{}, err
	}
	return *signed, nil
}

var _ promise.Issuer = LocalIssuer{}

// this is ugly adapter to make identity.Signer from node usable in payments package
// it's a bit confusing as both interfaces has the same name and method, but only params and return values differ
type paymentsSignerAdapter struct {
	identitySigner identity.Signer
}

func (psa paymentsSignerAdapter) Sign(data ...[]byte) ([]byte, error) {
	var message []byte
	for _, dataSlice := range data {
		message = append(message, dataSlice...)
	}

	sig, err := psa.identitySigner.Sign(message)
	if err != nil {
		return nil, err
	}
	return sig.Bytes(), nil
}

var _ payments_identity.Signer = paymentsSignerAdapter{}
