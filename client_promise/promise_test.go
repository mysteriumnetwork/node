package client_promise

import (
	dto "github.com/mysterium/node/client_promise/dto"
	discovery_dto "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ClientPromise(t *testing.T) {

	amount := discovery_dto.Money{
		Amount:   uint64(5),
		Currency: "Token",
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
	assert.Equal(t, promise.Amount.Currency, "Token")
}
