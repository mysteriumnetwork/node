package dto

import (
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/money"
)

const PAYMENT_METHOD_PER_BYTES = "PER_BYTES"

type PaymentMethodPerBytes struct {
	Price money.Money `json:"price"`

	// Service bytes provided for paid price
	Bytes datasize.BitSize `json:"bytes,omitempty"`
}

func (method PaymentMethodPerBytes) GetPrice() money.Money {
	return method.Price
}
