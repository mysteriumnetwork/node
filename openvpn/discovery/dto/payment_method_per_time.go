package dto

import (
	"github.com/mysterium/node/money"
	"time"
)

const PaymentMethodPerTime = "PER_TIME"

type PaymentPerTime struct {
	Price money.Money `json:"price"`

	// Service duration provided for paid price
	Duration time.Duration `json:"duration"`
}

func (method PaymentPerTime) GetPrice() money.Money {
	return method.Price
}
