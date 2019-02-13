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

package validators

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
)

// IssuedPromiseValidator validates issued promises
type IssuedPromiseValidator struct {
	consumer common.Address
	receiver common.Address
	issuer   common.Address
}

// NewIssuedPromiseValidator return a new instance of IssuedPromiseValidator
func NewIssuedPromiseValidator(consumer, receiver, issuer identity.Identity) *IssuedPromiseValidator {
	return &IssuedPromiseValidator{
		consumer: common.HexToAddress(consumer.Address),
		receiver: common.HexToAddress(receiver.Address),
		issuer:   common.HexToAddress(issuer.Address),
	}
}

// Validate checks if the issued promise is valid or not
func (ipv *IssuedPromiseValidator) Validate(promiseMsg promise.Message) bool {
	issuedPromise := promises.IssuedPromise{
		Promise: promises.Promise{
			Extra: promise.ExtraData{
				ConsumerAddress: ipv.consumer,
			},
			Amount:   promiseMsg.Amount,
			SeqNo:    promiseMsg.SequenceID,
			Receiver: ipv.receiver,
		},
		IssuerSignature: common.FromHex(promiseMsg.Signature),
	}

	issuer, err := issuedPromise.IssuerAddress()
	if err != nil {
		return false
	}
	return issuer == ipv.issuer
}
