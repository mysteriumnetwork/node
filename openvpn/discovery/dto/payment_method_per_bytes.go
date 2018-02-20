package dto

import (
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/money"
)

const PaymentMethodPerBytes = "PER_BYTES"

type PaymentPerBytes struct {
	Price money.Money `json:"price"`

	// Service bytes provided for paid price
	Bytes datasize.BitSize `json:"bytes,omitempty"`
}

func (method PaymentPerBytes) GetPrice() money.Money {
	return method.Price
}
