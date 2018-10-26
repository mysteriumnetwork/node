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
			"Amount": {
				"amount": 12500000,
				"currency": "MYST"
			}
		},
		"IssuerSignature": "sXeJmBrjjMXkGraD8ItqNEI+0IommozG4dF24FvgbWQdgHJi10EvJOLt0F2AS5Y0MBe8hlDo3B0vlH2hW5uHSQE="
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
	assert.Equal(t, response, responseInvalidPromise)
}

func TestConsumeBadSignature(t *testing.T) {
	consumer := Consumer{}
	signedPromise := &SignedPromise{Promise: Promise{}, IssuerSignature: "ProducerSignature"}
	request := &Request{signedPromise}
	response, err := consumer.Consume(request)
	assert.Equal(t, errBadSignature, err)
	assert.Equal(t, responseInvalidPromise, response)
}

func TestConsumeUnknownBenefiter(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	consumer := Consumer{}
	response, err := consumer.Consume(&request)
	assert.Equal(t, errUnknownBenefiter, err)
	assert.Equal(t, responseInvalidPromise, response)

}

func TestConsumeLowAmount(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{999999999},
	}
	consumer := Consumer{proposal: proposal, balance: fakeBlockchain(12500000)}
	response, err := consumer.Consume(&request)
	assert.Equal(t, errLowAmount, err)
	assert.Equal(t, responseInvalidPromise, response)
}

func TestConsumeLowBalance(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, balance: fakeBlockchain(1)}
	response, err := consumer.Consume(&request)
	assert.Equal(t, errLowBalance, err)
	assert.Equal(t, responseInvalidPromise, response)

}

func TestConsume(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x1526273ac60cdebfa2aece92da3261ecb564763a",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, balance: fakeBlockchain(999999999), storage: &fakeStorage{}}
	response, err := consumer.Consume(&request)
	assert.NoError(t, err)
	assert.Equal(t, &Response{Success: true}, response)
}

type fakePayment struct {
	amount uint64
}

func (fp fakePayment) GetPrice() money.Money {
	return money.Money{Amount: fp.amount}
}

func fakeBlockchain(balance uint64) identity.Balance {
	return func(_ identity.Identity) (uint64, error) {
		return balance, nil
	}
}

type fakeStorage struct{}

func (fs *fakeStorage) Store(issuer string, data interface{}) error  { return nil }
func (fs *fakeStorage) Delete(issuer string, data interface{}) error { return nil }
func (fs *fakeStorage) Close() error                                 { return nil }
func (fs *fakeStorage) StoreSession(bucketName string, key string, value interface{}) error {
	return nil
}
func (fs *fakeStorage) GetAll(issuer string, data interface{}) error { return nil }
func (fs *fakeStorage) Save(data interface{}) error                  { return nil }
func (fs *fakeStorage) Update(data interface{}) error                { return nil }
func (fs *fakeStorage) GetAllSessions(data interface{}) error        { return nil }
