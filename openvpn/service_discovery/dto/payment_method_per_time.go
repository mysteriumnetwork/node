package dto

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"time"
)

const PAYMENT_METHOD_PER_TIME = "PER_TIME"

type PaymentMethodPerTime struct {
	Price dto_discovery.Money

	// Service duration provided for paid price
	Duration time.Duration
}

func (method PaymentMethodPerTime) GetPrice() dto_discovery.Money {
	return method.Price
}
