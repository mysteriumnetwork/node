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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/node/config/remote"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func (c *cliApp) order(args []string) (err error) {
	var usage = strings.Join([]string{
		"Usage: order <action> [args]",
		"Available actions:",
		"  " + usageOrderCurrencies,
		"  " + usageOrderCreate,
		"  " + usageOrderGet,
		"  " + usageOrderGetAll,
	}, "\n")

	if len(args) == 0 {
		clio.Info(usage)
		return errWrongArgumentCount
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "create":
		return c.orderCreate(actionArgs)
	case "get":
		return c.orderGet(actionArgs)
	case "get-all":
		return c.orderGetAll(actionArgs)
	case "currencies":
		return c.currencies(actionArgs)
	default:
		fmt.Println(usage)
		return errUnknownSubCommand(args[0])
	}
}

const usageOrderCurrencies = "currencies"

func (c *cliApp) currencies(args []string) (err error) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageOrderCurrencies)
		return
	}

	resp, err := c.tequilapi.OrderCurrencies()
	if err != nil {
		return fmt.Errorf("could not get currencies: %w", err)
	}

	clio.Info(fmt.Sprintf("Supported currencies: %s", strings.Join(resp, ", ")))
	return nil
}

const usageOrderCreate = "create <identity> <amount> <pay currency> [use lightning network]"

func (c *cliApp) orderCreate(args []string) (err error) {
	if len(args) > 4 || len(args) < 3 {
		clio.Info("Usage: " + usageOrderCreate)
		return errWrongArgumentCount
	}

	f, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("could not parse amount: %w", err)
	}
	if f <= 0 {
		return errors.New("top up amount is required and must be greater than 0")
	}

	options, err := c.tequilapi.PaymentOptions()
	if err != nil {
		clio.Info("Failed to get payment options, wont check minimum possible amount to topup")
	}

	if options.Minimum != 0 && f <= options.Minimum {
		return fmt.Errorf(
			"top up amount must be greater than %v%s",
			options.Minimum,
			config.GetString(config.FlagDefaultCurrency))
	}

	ln := false
	if len(args) == 4 {
		b, err := strconv.ParseBool(args[3])
		if err != nil {
			return fmt.Errorf("[use lightning network]: only true/false allowed: %w", err)
		}
		ln = b
	}

	resp, err := c.tequilapi.OrderCreate(identity.FromAddress(args[0]), contract.OrderRequest{
		MystAmount:       f,
		PayCurrency:      args[2],
		LightningNetwork: ln,
	})
	if err != nil {
		return fmt.Errorf("could not create an order: %w", err)
	}
	printOrder(resp, c.config)
	return nil
}

const usageOrderGet = "get <identity> <orderID>"

func (c *cliApp) orderGet(args []string) (err error) {
	if len(args) != 2 {
		clio.Info("Usage: " + usageOrderGet)
		return
	}

	u, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse orderID: %w", err)
	}
	resp, err := c.tequilapi.OrderGet(identity.FromAddress(args[0]), u)
	if err != nil {
		return fmt.Errorf("could not get an order: %w", err)
	}
	printOrder(resp, c.config)
	return nil
}

const usageOrderGetAll = "get-all <identity>"

func (c *cliApp) orderGetAll(args []string) (err error) {
	if len(args) != 1 {
		clio.Info("Usage: " + usageOrderGetAll)
		return errWrongArgumentCount
	}

	resp, err := c.tequilapi.OrderGetAll(identity.FromAddress(args[0]))
	if err != nil {
		return fmt.Errorf("could not get orders: %w", err)
	}

	if len(resp) == 0 {
		clio.Info("No orders found")
		return nil
	}

	for _, r := range resp {
		clio.Info(fmt.Sprintf("Order ID '%d' is in state: '%s'", r.ID, r.Status))
	}
	clio.Info(
		fmt.Sprintf("To explore additional order information use: '%s'", usageOrderGet),
	)
	return nil
}

func printOrder(o contract.OrderResponse, rc *remote.Config) {
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
	clio.Info(fmt.Sprintf("Receive %s amount: %f", rc.GetStringByFlag(config.FlagDefaultCurrency), o.MystAmount))
	clio.Info(fmt.Sprintf("PaymentURL: %s", o.PaymentURL))
}
