package session

import (
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
)

type AmountCalc struct {
	PaymentDef dto.PaymentPerTime
}

func (ac AmountCalc) TotalAmount(duration time.Duration) money.Money {
	// time.Duration holds info in nanoseconds internally anyway (with max duration of 290 years) so we are probably safe here
	// however - careful testing of corner cases is needed
	// another question - in case of amount of 15 seconds, and price 10 myst per minute, total amount will be rounded to zero
	// add 1 in case it's bad
	amountInUnits := uint64(duration / ac.PaymentDef.Duration)

	return money.Money{
		Amount:   amountInUnits * ac.PaymentDef.Price.Amount,
		Currency: ac.PaymentDef.Price.Currency,
	}
}
