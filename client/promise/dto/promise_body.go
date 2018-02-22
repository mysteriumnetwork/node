package dto

import (
	"github.com/mysterium/node/money"
)

// PromiseBody represents payment promise between two parties
type PromiseBody struct {
	SerialNumber int
	IssuerID     string
	BenefiterID  string
	Amount       money.Money
}
