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

package account

import (
	"fmt"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func contains(needle string, stack []string) bool {
	for _, s := range stack {
		if needle == s {
			return true
		}
	}
	return false
}

func printOrder(o contract.OrderResponse) {
	strUnknown := func(s *string) string {
		if s == nil {
			return "unknown"
		}
		return *s
	}

	fUnknown := func(f *float64) string {
		if f == nil {
			return "unknown"
		}
		return fmt.Sprint(*f)
	}

	clio.Info(fmt.Sprintf("Order ID '%d' is in state: '%s'", o.ID, o.Status))
	clio.Info(fmt.Sprintf("Price: %s %s", fUnknown(o.PriceAmount), o.PriceCurrency))
	clio.Info(fmt.Sprintf("Pay: %s %s", fUnknown(o.PayAmount), strUnknown(o.PayCurrency)))
	clio.Info(fmt.Sprintf("Receive: %s %s", fUnknown(o.ReceiveAmount), o.ReceiveCurrency))
	clio.Info(fmt.Sprintf("Myst amount: %f", o.MystAmount))
	clio.Info(fmt.Sprintf("PaymentURL: %s", o.PaymentURL))
}
