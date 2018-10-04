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
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var jsonRequest = []byte(`{
	"SignedPromise": {
		"Promise": {
			"SerialNumber": 1,
			"IssuerID": "0x8eaf2780c6098dd1baee2b7c8c62f2d92ba1fe29",
			"BenefiterID": "0x67c8afcfc5432cf1ce8d5e289aeacbffa1904f82",
			"Amount": {
				"amount": 100,
				"currency": "MYST"
			}
		},
		"IssuerSignature": "GK/or0Ecwdatka4mpZipIC1+LBXVXdVNHKpkvUTo/2F4plqzTehRGDEfg74wzdtXN2uXdvoeUZC/aVE5Ine8sQE="
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
	assert.Nil(t, response)
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
		Request: request,
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
		Request: &request,
	}, response)
}

func TestConsumeLowAmount(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x67c8afcfc5432cf1ce8d5e289aeacbffa1904f82",
		PaymentMethod: fakePayment{999},
	}
	consumer := Consumer{proposal: proposal}
	response, err := consumer.Consume(&request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errLowAmount.Error(),
		Request: &request,
	}, response)
}

func TestConsumeLowBalance(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x67c8afcfc5432cf1ce8d5e289aeacbffa1904f82",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, etherClient: &fakeBlockchain{1}}
	response, err := consumer.Consume(&request)
	assert.Nil(t, err)
	assert.Equal(t, &Response{
		Success: false,
		Message: errLowBalance.Error(),
		Request: &request,
	}, response)
}

func TestConsume(t *testing.T) {
	var request Request
	err := json.Unmarshal(jsonRequest, &request)
	assert.Nil(t, err)

	proposal := dto.ServiceProposal{
		ProviderID:    "0x67c8afcfc5432cf1ce8d5e289aeacbffa1904f82",
		PaymentMethod: fakePayment{1},
	}
	consumer := Consumer{proposal: proposal, etherClient: &fakeBlockchain{99999}, storage: &fakeStorage{}}
	response, err := consumer.Consume(&request)
	assert.NoError(t, err)
	assert.Equal(t, &Response{
		Success: true,
		Message: "Promise accepted",
		Request: &request,
	}, response)
}

type fakePayment struct {
	amount uint64
}

func (fp fakePayment) GetPrice() money.Money {
	return money.Money{Amount: fp.amount}
}

type fakeBlockchain struct {
	balance int64
}

func (fb *fakeBlockchain) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return big.NewInt(fb.balance), nil
}
func (fb *fakeBlockchain) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}
func (fb *fakeBlockchain) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}
func (fb *fakeBlockchain) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return 0, nil
}

type fakeStorage struct{}

func (fs *fakeStorage) Store(issuer string, data interface{}) error  { return nil }
func (fs *fakeStorage) Delete(issuer string, data interface{}) error { return nil }
func (fs *fakeStorage) Close() error                                 { return nil }
