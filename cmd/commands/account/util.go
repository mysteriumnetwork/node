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
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/payments/crypto"
)

func findGateway(name string, gws []contract.GatewaysResponse) (*contract.GatewaysResponse, bool) {
	for _, gw := range gws {
		if gw.Name == name {
			return &gw, true
		}
	}
	return nil, false
}

func contains(needle string, stack []string) bool {
	for _, s := range stack {
		if needle == s {
			return true
		}
	}
	return false
}

func printOrder(o contract.PaymentOrderResponse) {
	clio.Info(fmt.Sprintf("Order ID '%s' is in state: '%s'", o.ID, o.Status))
	clio.Info(fmt.Sprintf("Pay: %f %s", o.PayAmount, o.PayCurrency))
	clio.Info(fmt.Sprintf("Receive: %s", money.New(crypto.FloatToBigMyst(o.ReceiveMYST)).String()))
	clio.Info("Data:", string(o.PublicGatewayData))
}
