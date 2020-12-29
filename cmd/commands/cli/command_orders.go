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

	remote_config "github.com/mysteriumnetwork/node/config/remote"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
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
		clio.Info(usage)
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
		clio.Warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
	}
}

const usageOrderCurrencies = "currencies"

func (c *cliApp) currencies(args []string) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageOrderCurrencies)
		return
	}

	resp, err := c.tequilapi.OrderCurrencies()
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not get currencies"))
		return
	}

	clio.Info(fmt.Sprintf("Supported currencies: %s", strings.Join(resp, ", ")))
}

const usageOrderCreate = "create <identity> <amount> <pay currency> [use lightning network]"

func (c *cliApp) orderCreate(args []string) {
	if len(args) > 4 || len(args) < 3 {
		clio.Info("Usage: " + usageOrderCreate)
		return
	}

	f, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		clio.Warn("could not parse amount")
		return
	}
	if f <= 0 {
		clio.Warn(fmt.Sprintf("Top up amount is required and must be greater than 0"))
		return
	}

	options, err := c.tequilapi.PaymentOptions()
	if err != nil {
		clio.Info("Failed to get payment options, wont check minimum possible amount to topup")
	}

	if options.Minimum != 0 && f <= options.Minimum {
		msg := fmt.Sprintf(
			"Top up amount must be greater than %v%s",
			options.Minimum,
			config.GetString(config.FlagDefaultCurrency))
		clio.Warn(msg)
		return
	}

	ln := false
	if len(args) == 4 {
		b, err := strconv.ParseBool(args[3])
		if err != nil {
			clio.Warn("[use lightning network]: only true/false allowed")
		}
		ln = b
	}

	resp, err := c.tequilapi.OrderCreate(identity.FromAddress(args[0]), contract.OrderRequest{
		MystAmount:       f,
		PayCurrency:      args[2],
		LightningNetwork: ln,
	})
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not create an order"))
		return
	}
	printOrder(resp)
}

const usageOrderGet = "get <identity> <orderID>"

func (c *cliApp) orderGet(args []string) {
	if len(args) != 2 {
		clio.Info("Usage: " + usageOrderGet)
		return
	}

	u, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		clio.Warn("could not parse orderID")
		return
	}
	resp, err := c.tequilapi.OrderGet(identity.FromAddress(args[0]), u)
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not get an order"))
		return
	}
	printOrder(resp)
}

const usageOrderGetAll = "get-all <identity>"

func (c *cliApp) orderGetAll(args []string) {
	if len(args) != 1 {
		clio.Info("Usage: " + usageOrderGetAll)
		return
	}

	resp, err := c.tequilapi.OrderGetAll(identity.FromAddress(args[0]))
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not get an orders"))
		return
	}

	if len(resp) == 0 {
		clio.Info("No orders found")
		return
	}

	for _, r := range resp {
		clio.Info(fmt.Sprintf("Order ID '%d' is in state: '%s'", r.ID, r.Status))
	}
	clio.Info(
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

	clio.Info(fmt.Sprintf("Order ID '%d' is in state: '%s'", o.ID, o.Status))
	clio.Info(fmt.Sprintf("Price: %s %s", fUnknown(o.PriceAmount), o.PriceCurrency))
	clio.Info(fmt.Sprintf("Pay: %s %s", fUnknown(o.PayAmount), strUnknown(o.PayCurrency)))
	clio.Info(fmt.Sprintf("Receive: %s %s", fUnknown(o.ReceiveAmount), o.ReceiveCurrency))
	clio.Info(fmt.Sprintf("Receive %s amount: %f", remote_config.Config.GetStringByFlag(config.FlagDefaultCurrency), o.MystAmount))
	clio.Info(fmt.Sprintf("PaymentURL: %s", o.PaymentURL))
}
