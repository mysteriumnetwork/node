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

package contract

import "github.com/mysteriumnetwork/node/pilvytis"

// PaymentOrderOptions represents pilvytis payment order options
// swagger:model PaymentOrderOptions
type PaymentOrderOptions struct {
	Minimum   float64   `json:"minimum"`
	Suggested []float64 `json:"suggested"`
}

// ToPaymentOrderOptions - convert pilvytis.PaymentOrderOptions to contract.ToPaymentOrderOptions
func ToPaymentOrderOptions(poo *pilvytis.PaymentOrderOptions) *PaymentOrderOptions {
	return &PaymentOrderOptions{
		Minimum:   poo.Minimum,
		Suggested: poo.Suggested,
	}
}
