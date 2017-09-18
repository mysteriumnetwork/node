package client_promise

import (
	"testing"
	"github.com/stretchr/testify/assert"
	dto "github.com/mysterium/node/client_promise/dto"
)

func Test_ClientPromise(t *testing.T) {
	promise := dto.PromiseBody{
		SerialNumber: 1,
		IssuerId: "issuer1",
		BenefiterId: "benefiter1",
		Amount: 5,
	}

	assert.Equal(t, promise.SerialNumber, 1)
	assert.Equal(t, promise.IssuerId, "issuer1")
	assert.Equal(t, promise.BenefiterId, "benefiter1")
	assert.Equal(t, promise.Amount, 5)
}