package dto

import "github.com/mysterium/node/money"

type PaymentMethod interface {
	// Service price per unit of metering
	GetPrice() money.Money
}
