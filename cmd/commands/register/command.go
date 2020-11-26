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

package register

import (
	"math/big"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity/registry"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"

	"github.com/urfave/cli/v2"
)

// CommandName is the name of register command.
const CommandName = "register"

// NewCommand function creates license command.
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:   CommandName,
		Usage:  "Register a new identity",
		Before: clicontext.LoadUserConfigQuietly,
		Action: func(ctx *cli.Context) error {
			config.ParseFlagsNode(ctx)
			nodeOptions := node.GetOptions()
			cmd := &command{
				tequilapi: tequilapi_client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort),
			}
			cmd.run()

			return nil
		},
	}
}

type command struct {
	tequilapi *tequilapi_client.Client
}

func (c *command) run() {
	id, err := c.tequilapi.CurrentIdentity("", "")
	if err != nil {
		clio.Warn("Could not get or create identity")
	}

	identityStatus, err := c.tequilapi.Identity(id.Address)
	if err != nil {
		clio.Warn("Failed to get identity status")
		return
	}

	if identityStatus.RegistrationStatus != registry.Unregistered.String() {
		clio.Infof("Already have an identity: %s\n", id.Address)
		return
	}

	if err := c.tequilapi.Unlock(id.Address, ""); err != nil {
		clio.Warn("Failed to unlock the identity")
		return
	}

	c.registerIdentity(id.Address)
}

func (c *command) registerIdentity(identity string) {
	fees, err := c.tequilapi.GetTransactorFees()
	if err != nil {
		clio.Error("Failed to get fees for registration")
		return
	}

	err = c.tequilapi.RegisterIdentity(identity, "", new(big.Int).SetInt64(0), fees.Registration, nil)
	if err != nil {
		clio.Error("Failed to register the identity")
		return
	}

	msg := "Registration started. Topup the identities channel to finish it."
	if config.GetBool(config.FlagTestnet2) || config.GetBool(config.FlagTestnet) {
		msg = "Registration successful, try to connect."
	}
	clio.Success(msg)
}
