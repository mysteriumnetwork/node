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

package promise

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var jsonRequest = []byte(`{
	"SignedPromise": {
		"Promise": {
			"SerialNumber": 1,
			"IssuerID": "0x8eaf2780c6098dd1baee2b7c8c62f2d92ba1fe29",
			"BenefiterID": "0x1526273ac60cdebfa2aece92da3261ecb564763a",
			"Fee": {
				"amount": 12500000,
				"currency": "MYST"
			}
		},
		"IssuerSignature": "rjBYE1rglsb3UVeIplTqodA4mgowkpNoxz89rZTwVf4KsDPhwx0RfyREd86wbZpXTnxs6Ry9rixOqpjOs4iAawE="
	}
}`)

func TestNewRequest(t *testing.T) {
	consumer := Consumer{}

	assert.Equal(t, &Request{}, consumer.NewRequest())
}

func TestConsumeUnsupportedRequest(t *testing.T) {
	consumer := Consumer{}
	response, err := consumer.Consume(0)
	assert.Error(t, errUnsupportedRequest, err)
	assert.Equal(t, response, failedResponse(errUnsupportedRequest))
}

func TestConsumeBadSignature(t *testing.T) {
	consumer := Consumer{}
	signedPromise := &SignedPromise{Promise: Promise{}, IssuerSignature: "ProducerSignature"}
	request := &Request{signedPromise}
	response, err := consumer.Consume(request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errBadSignature.Error(),
	}, response)
}

func TestConsumeUnknownBenefiter(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	consumer := Consumer{}
	response, err := consumer.Consume(&request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errUnknownBenefiter.Error(),
	}, response)
}

func TestConsumeLowAmount(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{999999999},
	}
	consumer := Consumer{proposal: proposal, balanceRegistry: fakeBlockchain(12500000)}
	response, err := consumer.Consume(&request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errLowAmount.Error(),
	}, response)
}

func TestConsumeLowBalance(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, balanceRegistry: fakeBlockchain(1)}
	response, err := consumer.Consume(&request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errLowBalance.Error(),
	}, response)
}

func TestConsume(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, balanceRegistry: fakeBlockchain(999999999), storage: &fakeStorage{}}
	response, err := consumer.Consume(&request)
	assert.NoError(t, err)
	assert.Equal(t, &Response{
		Success: true,
		Message: "Promise accepted",
	}, response)
}

type fakePayment struct {
	amount uint64
}

func (fp fakePayment) GetPrice() money.Money {
	return money.Money{Amount: fp.amount}
}

func fakeBlockchain(balance uint64) identity.BalanceRegistry {
	return func(_ identity.Identity) (uint64, error) {
		return balance, nil
	}
}

type fakeStorage struct{}

func (fs *fakeStorage) Store(issuer string, data interface{}) error  { return nil }
func (fs *fakeStorage) Delete(issuer string, data interface{}) error { return nil }
func (fs *fakeStorage) Close() error                                 { return nil }
