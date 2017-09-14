package client_promise

import (
	"testing"
	"github.com/stretchr/testify/assert"
	dto "github.com/mysterium/node/client_promise/dto"
)

func Test_ClientPromise(t *testing.T) {
	promise := dto.ClientPromise{
		SerialNumber: 1,
		IssuerId: 1,
		BenefiterId: 1,
		Amount: 5,
	}

	assert.Equal(t, promise.SerialNumber, 1)
	assert.Equal(t, promise.IssuerId, 1)
	assert.Equal(t, promise.BenefiterId, 1)
	assert.Equal(t, promise.Amount, 5)
}