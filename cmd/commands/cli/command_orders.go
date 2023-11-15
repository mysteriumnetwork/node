/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/payments/exchange"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func (c *cliApp) order(args []string) (err error) {
	var usage = strings.Join([]string{
		"Usage: order <action> [args]",
		"Available actions:",
		"  " + usageOrderGateways,
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
	case "invoice":
		return c.invoice(actionArgs)
	case "gateways":
		return c.gateways(actionArgs)
	default:
		fmt.Println(usage)
		return errUnknownSubCommand(args[0])
	}
}

const usageOrderGateways = "gateways"

func (c *cliApp) gateways(args []string) (err error) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageOrderGateways)
		return
	}

	resp, err := c.tequilapi.PaymentOrderGateways(exchange.CurrencyMYST)
	if err != nil {
		return fmt.Errorf("could not get currencies: %w", err)
	}

	for _, gw := range resp {
		clio.Info("Gateway:", gw.Name)
		clio.Info("Suggested minimum order:", gw.OrderOptions.Minimum)
		clio.Info("Supported currencies:", strings.Join(gw.Currencies, ", "))
	}

	return nil
}

const usageOrderCreate = "create <identity> <amount> <pay currency> <gateway> <country> [gw data: lightning_network=true,order=123]"

func (c *cliApp) orderCreate(args []string) (err error) {
	if len(args) != 5 && len(args) != 6 {
		clio.Info("Usage: " + usageOrderCreate)
		return errWrongArgumentCount
	}

	argId := args[0]
	argAmount := args[1]
	argPayCurrency := args[2]
	argGateway := args[3]
	argCountry := args[4]
	var argCallerData string
	if len(args) > 5 {
		argCallerData = args[5]
	}

	f, err := strconv.ParseFloat(argAmount, 64)
	if err != nil {
		return fmt.Errorf("could not parse amount: %w", err)
	}
	if f <= 0 {
		return errors.New("top-up amount is required and must be greater than 0")
	}

	gws, err := c.tequilapi.PaymentOrderGateways(exchange.CurrencyMYST)
	if err != nil {
		clio.Info("Failed to get enabled gateways and their information")
		return
	}

	if len(gws) == 0 {
		clio.Warn("No payment gateways are enabled, can't create new orders")
		return
	}

	gw, ok := findGateway(argGateway, gws)
	if !ok {
		clio.Error("Can't continue, no such gateway:", argGateway)
		return
	}
	if gw.OrderOptions.Minimum != 0 && f <= gw.OrderOptions.Minimum {
		return fmt.Errorf(
			"top-up amount must be greater than %v%s",
			gw.OrderOptions.Minimum,
			config.GetString(config.FlagDefaultCurrency))
	}

	data := map[string]interface{}{}
	parts := strings.Split(argCallerData, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			clio.Error("Gateway data wrong, example: lightning_network=true,custom_id=\"123 11\"")
			return
		}

		if b, err := strconv.ParseBool(kv[1]); err == nil {
			data[kv[0]] = b
			continue
		}

		if b, err := strconv.ParseFloat(kv[1], 64); err == nil {
			data[kv[0]] = b
			continue
		}

		data[kv[0]] = kv[1]
	}

	callerData, err := json.Marshal(data)
	if err != nil {
		clio.Error("Failed to make caller data")
		return
	}

	resp, err := c.tequilapi.OrderCreate(
		identity.FromAddress(argId),
		argGateway,
		contract.PaymentOrderRequest{
			MystAmount:  argAmount,
			PayCurrency: argPayCurrency,
			Country:     argCountry,
			CallerData:  callerData,
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

	resp, err := c.tequilapi.OrderGet(identity.FromAddress(args[0]), args[1])
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
		clio.Info(fmt.Sprintf("Order ID '%s' is in state: '%s'", r.ID, r.Status))
	}
	clio.Info(
		fmt.Sprintf("To explore additional order information use: orders '%s'", usageOrderGet),
	)
	return nil
}

const usageOrderInvoice = "invoice <identity> <orderID>"

func (c *cliApp) invoice(args []string) (err error) {
	if len(args) != 2 {
		clio.Info("Usage: " + usageOrderInvoice)
		return
	}

	resp, err := c.tequilapi.OrderInvoice(identity.FromAddress(args[0]), args[1])
	if err != nil {
		return fmt.Errorf("could not get an order invoice: %w", err)
	}
	filename := fmt.Sprintf("invoice-%v.pdf", args[1])
	clio.Info("Writing invoice to", filename)
	return os.WriteFile(filename, resp, 0644)
}

func printOrder(o contract.PaymentOrderResponse, rc *remote.Config) {
	clio.Info(fmt.Sprintf("Order ID '%s' is in state: '%s'", o.ID, o.Status))
	clio.Info(fmt.Sprintf("Pay: %s %s", o.PayAmount, o.PayCurrency))
	clio.Info(fmt.Sprintf("Receive MYST: %s", o.ReceiveMYST))
	clio.Info("Order details:")
	clio.Info(fmt.Sprintf(" + MYST (%v): %s %s", o.ReceiveMYST, o.ItemsSubTotal, o.Currency))
	clio.Info(fmt.Sprintf(" + Tax (%s %%): %s %s", o.TaxRate, o.TaxSubTotal, o.Currency))
	clio.Info(fmt.Sprintf(" = Total: %s %s", o.OrderTotal, o.Currency))
	clio.Info("Data:", string(o.PublicGatewayData))
}

func findGateway(name string, gws []contract.GatewaysResponse) (*contract.GatewaysResponse, bool) {
	for _, gw := range gws {
		if gw.Name == name {
			return &gw, true
		}
	}
	return nil, false
}
