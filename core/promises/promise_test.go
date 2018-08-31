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

package promises

import (
	"testing"

	dto "github.com/mysterium/node/core/promises/dto"
	"github.com/mysterium/node/money"
	"github.com/stretchr/testify/assert"
)

const CurrencyToken = money.Currency("Token")

func Test_PromiseBody(t *testing.T) {

	amount := money.Money{
		Amount:   uint64(5),
		Currency: CurrencyToken,
	}

	promise := dto.PromiseBody{
		SerialNumber: 1,
		IssuerID:     "issuer1",
		BenefiterID:  "benefiter1",
		Amount:       amount,
	}

	assert.Equal(t, promise.SerialNumber, 1)
	assert.Equal(t, promise.IssuerID, "issuer1")
	assert.Equal(t, promise.BenefiterID, "benefiter1")
	assert.Equal(t, promise.Amount.Amount, uint64(5))
	assert.Equal(t, promise.Amount.Currency, CurrencyToken)
}

func Test_SignedPromise(t *testing.T) {

	promise := dto.PromiseBody{}

	signedPromise := dto.SignedPromise{
		Promise:         promise,
		IssuerSignature: "signature",
	}

	assert.Equal(t, signedPromise.Promise, promise)
	assert.Equal(t, signedPromise.IssuerSignature, dto.Signature("signature"))
}
