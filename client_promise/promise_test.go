package client_promise

import (
	dto "github.com/mysterium/node/client_promise/dto"
	"github.com/mysterium/node/money"
	"github.com/stretchr/testify/assert"
	"testing"
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
