package dto

import (
	"github.com/mysterium/node/datasize"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

const PAYMENT_METHOD_PER_BYTES = "PER_BYTES"

type PaymentMethodPerBytes struct {
	Price dto_discovery.Money

	// Service bytes provided for paid price
	Bytes datasize.BitSize
}

func (method PaymentMethodPerBytes) GetPrice() dto_discovery.Money {
	return method.Price
}
