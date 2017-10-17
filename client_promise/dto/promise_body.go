package dto

import (
	"github.com/mysterium/node/money"
)

type PromiseBody struct {
	SerialNumber int
	IssuerId     string
	BenefiterId  string
	Amount       money.Money
}