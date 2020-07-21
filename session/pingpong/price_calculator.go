/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/rs/zerolog/log"
)

func isServiceFree(method market.PaymentMethod) bool {
	if method == nil {
		return true
	}

	if method.GetPrice().Amount.Cmp(big.NewInt(0)) == 0 {
		return true
	}

	if method.GetRate().PerByte == 0 && method.GetRate().PerTime == 0 {
		return true
	}

	return false
}

// CalculatePaymentAmount calculates the required payment amount.
func CalculatePaymentAmount(timePassed time.Duration, bytesTransferred DataTransferred, method market.PaymentMethod) *big.Int {
	if isServiceFree(method) {
		return new(big.Int)
	}

	var ticksPassed float64
	price := method.GetPrice().Amount

	// avoid division by zero on free service
	if method.GetRate().PerTime > 0 {
		ticksPassed = float64(timePassed) / float64(method.GetRate().PerTime)
	}

	ticks := big.NewFloat(ticksPassed)
	timeComponent := new(big.Float).Mul(ticks, new(big.Float).SetInt(price))

	var chunksTransferred float64
	if method.GetRate().PerByte > 0 {
		chunksTransferred = float64(bytesTransferred.sum()) / float64(method.GetRate().PerByte)
	}

	chunks := big.NewFloat(chunksTransferred)
	byteComponent := new(big.Float).Mul(chunks, new(big.Float).SetInt(price))
	tc, _ := timeComponent.Int(nil)
	bc, _ := byteComponent.Int(nil)

	total := new(big.Int).Add(tc, bc)
	log.Debug().Msgf("Calculated price %v. Time component: %v, data component: %v ", total, timeComponent, byteComponent)
	return total
}
