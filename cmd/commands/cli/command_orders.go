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

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/pkg/errors"
)

func (c *cliApp) order(argsString string) {
	var usage = strings.Join([]string{
		"Usage: order <action> [args]",
		"Available actions:",
		"  " + usageOrderCurrencies,
		"  " + usageOrderCreate,
		"  " + usageOrderGet,
		"  " + usageOrderGetAll,
	}, "\n")

	if len(argsString) == 0 {
		info(usage)
		return
	}

	args := strings.Fields(argsString)
	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "create":
		c.orderCreate(actionArgs)
	case "get":
		c.orderGet(actionArgs)
	case "get-all":
		c.orderGetAll(actionArgs)
	case "currencies":
		c.currencies(actionArgs)
	default:
		warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
	}
}

const usageOrderCurrencies = "currencies"

func (c *cliApp) currencies(args []string) {
	if len(args) > 0 {
		info("Usage: " + usageOrderCurrencies)
		return
	}

	resp, err := c.tequilapi.OrderCurrencies()
	if err != nil {
		warn(errors.Wrap(err, "could not get currencies"))
		return
	}

	info(fmt.Sprintf("Supported currencies: %s", strings.Join(resp, ", ")))
}

const usageOrderCreate = "create <identity> <amount> <pay currency> [use lightning network]"

func (c *cliApp) orderCreate(args []string) {
	if len(args) > 4 || len(args) < 3 {
		info("Usage: " + usageOrderCreate)
		return
	}

	f, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		warn("could not parse amount")
		return
	}

	ln := false
	if len(args) == 4 {
		b, err := strconv.ParseBool(args[3])
		if err != nil {
			warn("[use lightning network]: only true/false allowed")
		}
		ln = b
	}

	resp, err := c.tequilapi.OrderCreate(identity.FromAddress(args[0]), contract.OrderRequest{
		MystAmount:       f,
		PayCurrency:      args[2],
		LightningNetwork: ln,
	})
	if err != nil {
		warn(errors.Wrap(err, "could not create an order"))
		return
	}
	printOrder(resp)
}

const usageOrderGet = "get <identity> <orderID>"

func (c *cliApp) orderGet(args []string) {
	if len(args) != 2 {
		info("Usage: " + usageOrderGet)
		return
	}

	u, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		warn("could not parse orderID")
		return
	}
	resp, err := c.tequilapi.OrderGet(identity.FromAddress(args[0]), u)
	if err != nil {
		warn(errors.Wrap(err, "could not get an order"))
		return
	}
	printOrder(resp)
}

const usageOrderGetAll = "get-all <identity>"

func (c *cliApp) orderGetAll(args []string) {
	if len(args) != 1 {
		info("Usage: " + usageOrderGetAll)
		return
	}

	resp, err := c.tequilapi.OrderGetAll(identity.FromAddress(args[0]))
	if err != nil {
		warn(errors.Wrap(err, "could not get an orders"))
		return
	}

	if len(resp) == 0 {
		info("No orders found")
		return
	}

	for _, r := range resp {
		info(fmt.Sprintf("Order ID '%d' is in state: '%s'", r.ID, r.Status))
	}
	info(
		fmt.Sprintf("To explore additional order information use: '%s'", usageOrderGet),
	)
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

	info(fmt.Sprintf("Order ID '%d' is in state: '%s'", o.ID, o.Status))
	info(fmt.Sprintf("Price: %s %s", fUnknown(o.PriceAmount), o.PriceCurrency))
	info(fmt.Sprintf("Pay: %s %s", fUnknown(o.PayAmount), strUnknown(o.PayCurrency)))
	info(fmt.Sprintf("Receive: %s %s", fUnknown(o.ReceiveAmount), o.ReceiveCurrency))
	info(fmt.Sprintf("Myst amount: %f", o.MystAmount))
	info(fmt.Sprintf("PaymentURL: %s", o.PaymentURL))
}
