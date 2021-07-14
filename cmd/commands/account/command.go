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
	"errors"
	"fmt"
	"strings"

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
		Usage: "Amount of MYST you want to top up in to your account",
	}

	flagCurrency = cli.StringFlag{
		Name:  "currency",
		Usage: "Currency you want to use when paying for your top up",
	}

	flagLastTopup = cli.BoolFlag{
		Name:  "last-topup",
		Usage: "Include last top up information",
	}

	flagToken = cli.StringFlag{
		Name:  "token",
		Usage: "Either a referral or affiliate token which can be used when registering",
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
				Usage: "Create a new top up for your account",
				Flags: []cli.Flag{&flagAmount, &flagCurrency},
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
		clio.Status("SECTION", "Last topup information:")
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

	currencies, err := c.tequilapi.OrderCurrencies()
	if err != nil {
		clio.Error("Could not get a list of supported currencies")
		return
	}

	currency := ctx.String(flagCurrency.Name)
	if !contains(currency, currencies) {
		clio.Warn("Given currency cannot be used")
		clio.Info("Supported currencies are:", strings.Join(currencies, ", "))
		return
	}

	amount := ctx.Float64(flagAmount.Name)
	if amount <= 0 {
		clio.Warn("Top up amount is required and must be greater than 0")
		return
	}

	options, err := c.tequilapi.PaymentOptions()
	if err != nil {
		clio.Info("Failed to get payment options, wont check minimum possible amount to topup")
	}

	if options.Minimum != 0 && amount <= options.Minimum {
		msg := fmt.Sprintf(
			"Top up amount must be greater than %v%s",
			options.Minimum,
			c.cfg.GetStringByFlag(config.FlagDefaultCurrency))
		clio.Warn(msg)
		return
	}

	resp, err := c.tequilapi.OrderCreate(identity.FromAddress(id.Address), contract.OrderRequest{
		MystAmount:       amount,
		PayCurrency:      currency,
		LightningNetwork: false,
	})
	if err != nil {
		clio.Error("Failed to create an top up request, make sure your requested amount is equal or more than 0.0001 BTC")
		return
	}

	clio.Info(fmt.Sprintf("New top up request for identity %s has been created: ", id.Address))
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
	err := c.tequilapi.RegisterIdentity(identity, token)
	if err != nil {
		clio.Error("Failed to register the identity")
		return
	}

	msg := "Registration started. Topup the identities channel to finish it."
	if c.cfg.GetBoolByFlag(config.FlagTestnet3) {
		msg = "Registration successful, try to connect."
	}
	clio.Success(msg)
}

func (c *command) identityIsUnregistered(identityAddress string) (bool, error) {
	identityStatus, err := c.tequilapi.Identity(identityAddress)
	if err != nil {
		return false, errors.New("Failed to get identity status")
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
		clio.Error("Failed to get topup information")
		return
	}

	if len(resp) == 0 {
		clio.Info("You have no topup requests")
		return
	}
	printOrder(resp[len(resp)-1])
}
