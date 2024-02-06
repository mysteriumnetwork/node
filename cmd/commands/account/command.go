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

package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/payments/exchange"
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/money"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

// CommandName is the name of the main command.
const CommandName = "account"

var (
	flagAmount = cli.Float64Flag{
		Name:  "amount",
		Usage: "Amount of MYST for the top-up",
	}

	flagCurrency = cli.StringFlag{
		Name:  "currency",
		Usage: "Currency you want to use when paying for your top-up",
	}

	flagGateway = cli.StringFlag{
		Name:  "gateway",
		Usage: "Gateway to use",
	}

	flagGwData = cli.StringFlag{
		Name:  "data",
		Usage: "Data required to use the gateway",
	}

	flagLastTopup = cli.BoolFlag{
		Name:  "last-topup",
		Usage: "Include last top-up information",
	}

	flagToken = cli.StringFlag{
		Name:  "token",
		Usage: "Either a referral or affiliate token which can be used when registering",
	}

	flagCountry = cli.StringFlag{
		Name:  "country",
		Usage: "Country code",
	}

	flagState = cli.StringFlag{
		Name:  "state",
		Usage: "State code",
	}
)

// NewCommand function creates license command.
func NewCommand() *cli.Command {
	var cmd *command

	return &cli.Command{
		Name:        CommandName,
		Usage:       "Manage your account",
		Description: "Using account subcommands you can manage your account details and get information about it",
		Flags:       []cli.Flag{&config.FlagTequilapiAddress, &config.FlagTequilapiPort},
		Before: func(ctx *cli.Context) error {
			tc, err := clio.NewTequilApiClient(ctx)
			if err != nil {
				return err
			}

			cfg, err := remote.NewConfig(tc)
			if err != nil {
				return err
			}

			cmd = &command{
				tequilapi: tc,
				cfg:       cfg,
			}
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "register",
				Usage: "Submit a registration request",
				Flags: []cli.Flag{&flagToken},
				Action: func(ctx *cli.Context) error {
					cmd.register(ctx)
					return nil
				},
			},
			{
				Name:  "topup",
				Usage: "Create a new top-up for your account",
				Flags: []cli.Flag{&flagAmount, &flagCurrency, &flagGateway, &flagGwData, &flagCountry, &flagState},
				Action: func(ctx *cli.Context) error {
					cmd.topup(ctx)
					return nil
				},
			},
			{
				Name:  "info",
				Usage: "Display information about identity account currently in use",
				Flags: []cli.Flag{&flagLastTopup},
				Action: func(ctx *cli.Context) error {
					cmd.info(ctx)
					return nil
				},
			},
			{
				Name:      "set-identity",
				Usage:     "Sets a new identity for your account which will be used in commands that require it",
				ArgsUsage: "[IdentityAddress]",
				Action: func(ctx *cli.Context) error {
					cmd.setIdentity(ctx)
					return nil
				},
			},
		},
	}
}

type command struct {
	tequilapi *tequilapi_client.Client
	cfg       *remote.Config
}

func (c *command) setIdentity(ctx *cli.Context) {
	givenID := ctx.Args().First()
	if givenID == "" {
		clio.Warn("No identity provided")
		return
	}

	if _, err := c.tequilapi.CurrentIdentity(givenID, ""); err != nil {
		clio.Error("Failed to set identity as default")
		return
	}

	clio.Success(fmt.Sprintf("Identity %s set as default", givenID))
}

func (c *command) info(ctx *cli.Context) {
	id, err := c.tequilapi.CurrentIdentity("", "")
	if err != nil {
		clio.Error("Failed to display information: could not get current identity")
		return
	}

	clio.Status("SECTION", "General account information:")
	c.infoGeneral(id.Address)

	if ctx.Bool(flagLastTopup.Name) {
		clio.Status("SECTION", "Last top-up information:")
		c.infoTopUp(id.Address)
	}
}

func (c *command) topup(ctx *cli.Context) {
	id, err := c.tequilapi.CurrentIdentity("", "")
	if err != nil {
		clio.Warn("Could not get your identity")
		return
	}

	ok, err := c.identityIsUnregistered(id.Address)
	if err != nil {
		clio.Error(err.Error())
		return
	}

	if ok {
		clio.Warn("Your identity is not registered, please execute `myst account register` first")
		return
	}

	gws, err := c.tequilapi.PaymentOrderGateways(exchange.CurrencyMYST)
	if err != nil {
		clio.Info("Failed to get enabled gateways and their information")
	}

	gatewayName := ctx.String(flagGateway.Name)

	gw, ok := findGateway(gatewayName, gws)
	if !ok {
		clio.Error("Can't continue, no such gateway:", gatewayName)
		return
	}

	amount := ctx.String(flagAmount.Name)

	amountF, err := strconv.ParseFloat(amount, 64)
	if err != nil || amountF <= 0 {
		clio.Warn("Top-up amount is required and must be greater than 0")
		return
	}

	if gw.OrderOptions.Minimum != 0 && amountF <= gw.OrderOptions.Minimum {
		clio.Error(
			fmt.Sprintf(
				"Top-up amount must be greater than %v%s",
				gw.OrderOptions.Minimum,
				config.GetString(config.FlagDefaultCurrency)))
		return
	}

	currency := ctx.String(flagCurrency.Name)
	if !contains(currency, gw.Currencies) {
		clio.Warn("Given currency cannot be used")
		clio.Info("Supported currencies are:", strings.Join(gw.Currencies, ", "))
		return
	}

	country := ctx.String(flagCountry.Name)
	if country == "" {
		clio.Warn("Country is required")
		return
	} else if len(country) != 2 {
		clio.Warn("Country code must be 2 characters long")
		return
	}

	state := ctx.String(flagState.Name)
	if state != "" && len(state) != 2 {
		clio.Warn("State code must be 2 characters long")
		return
	}

	callerData := json.RawMessage("{}")
	if gwData := ctx.String(flagGwData.Name); len(gwData) > 0 {
		data := map[string]interface{}{}
		parts := strings.Split(gwData, ",")
		for _, part := range parts {
			kv := strings.Split(part, "=")
			if len(kv) != 2 {
				clio.Error("Gateway data wrong, example: lightning_network=true,custom_id=123")
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

		callerData, err = json.Marshal(data)
		if err != nil {
			clio.Error("Failed to make caller data")
			return
		}
	}

	resp, err := c.tequilapi.OrderCreate(identity.FromAddress(id.Address), gatewayName, contract.PaymentOrderRequest{
		MystAmount:  amount,
		PayCurrency: currency,
		CallerData:  callerData,
		Country:     country,
		State:       state,
	})
	if err != nil {
		clio.Error("Failed to create a top-up request, make sure your requested amount is equal or more than 0.0001 BTC")
		return
	}

	clio.Info(fmt.Sprintf("New top-up request for identity %s has been created: ", id.Address))
	printOrder(resp)
}

func (c *command) register(ctx *cli.Context) {
	id, err := c.tequilapi.CurrentIdentity("", "")
	if err != nil {
		clio.Warn("Could not get or create identity")
		return
	}

	ok, err := c.identityIsUnregistered(id.Address)
	if err != nil {
		clio.Error(err.Error())
		return
	}
	if !ok {
		clio.Infof("Already have an identity: %s\n", id.Address)
		return
	}

	if err := c.tequilapi.Unlock(id.Address, ""); err != nil {
		clio.Warn("Failed to unlock the identity")
		return
	}

	c.registerIdentity(id.Address, c.parseToken(ctx))
}

func (c *command) parseToken(ctx *cli.Context) *string {
	if val := ctx.String(flagToken.Name); val != "" {
		return &val
	}

	return nil
}

func (c *command) registerIdentity(identity string, token *string) {
	err := c.tequilapi.RegisterIdentity(identity, "", token)
	if err != nil {
		clio.Error("Failed to register the identity")
		return
	}

	msg := "Registration started. Top up the identities channel to finish it."
	clio.Success(msg)
}

func (c *command) identityIsUnregistered(identityAddress string) (bool, error) {
	identityStatus, err := c.tequilapi.Identity(identityAddress)
	if err != nil {
		return false, errors.New("failed to get identity status")
	}

	return identityStatus.RegistrationStatus == registry.Unregistered.String(), nil
}

func (c *command) infoGeneral(identityAddress string) {
	identityStatus, err := c.tequilapi.Identity(identityAddress)
	if err != nil {
		clio.Error("Failed to display general account information: failed to fetch data")
		return
	}

	clio.Info("Using identity:", identityAddress)
	clio.Info("Registration Status:", identityStatus.RegistrationStatus)
	clio.Info("Channel address:", identityStatus.ChannelAddress)
	clio.Info(fmt.Sprintf("Balance: %s", money.New(identityStatus.Balance)))
}

func (c *command) infoTopUp(identityAddress string) {
	resp, err := c.tequilapi.OrderGetAll(identity.FromAddress(identityAddress))
	if err != nil {
		clio.Error("Failed to get top-up information")
		return
	}

	if len(resp) == 0 {
		clio.Info("You have no top-up requests")
		return
	}
	printOrder(resp[len(resp)-1])
}
