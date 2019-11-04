/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package session

import (
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session/pingpong/paydef"
)

// AmountCalc calculates the pay required given the amount
type AmountCalc struct {
	PaymentDef paydef.PaymentRate
}

// TotalAmount gets the total amount of money to pay given the duration
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
