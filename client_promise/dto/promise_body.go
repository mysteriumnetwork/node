package dto

import (
	"github.com/mysterium/node/money"
)

type PromiseBody struct {
	SerialNumber int
	IssuerID     string
	BenefiterID  string
	Amount       money.Money
}
