package client_promise

import (
	dto "github.com/mysterium/node/client_promise/dto"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/mysterium/node/money"
)

const CURRENCY_TOKEN = money.Currency("Token")

func Test_PromiseBody(t *testing.T) {

	amount := money.Money{
		Amount:   uint64(5),
		Currency: CURRENCY_TOKEN,
	}

	promise := dto.PromiseBody{
		SerialNumber: 1,
		IssuerId:     "issuer1",
		BenefiterId:  "benefiter1",
		Amount:       amount,
	}

	assert.Equal(t, promise.SerialNumber, 1)
	assert.Equal(t, promise.IssuerId, "issuer1")
	assert.Equal(t, promise.BenefiterId, "benefiter1")
	assert.Equal(t, promise.Amount.Amount, uint64(5))
	assert.Equal(t, promise.Amount.Currency, CURRENCY_TOKEN)
}

func Test_SignedPromise(t *testing.T) {

	promise := dto.PromiseBody{}

	signedPromise := dto.SignedPromise{
		Promise: promise,
		IssuerSignature: "signature",
	}

	assert.Equal(t, signedPromise.Promise, promise)
	assert.Equal(t, signedPromise.IssuerSignature, dto.Signature("signature"))
}