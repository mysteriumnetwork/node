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

package connection

import (
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"
	"github.com/mysteriumnetwork/node/core/node"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"

	"github.com/urfave/cli/v2"
)

// CommandName is the name of this command
const CommandName = "connection"

var (
	flagCountry = cli.StringFlag{
		Name:  "country",
		Usage: "Two letter (ISO 3166-1 alpha-2) country code to filter proposals.",
	}

	flagLocationType = cli.StringFlag{
		Name:  "location-type",
		Usage: "Node location types to filter by eg.'hosting', 'residential', 'mobile' etc.",
	}
)

const serviceWireguard = "wireguard"

// NewCommand function creates license command.
func NewCommand() *cli.Command {
	var cmd *command

	return &cli.Command{
		Name:        CommandName,
		Usage:       "Manage your connection",
		Description: "Using the connection subcommands you can manage your connection or get additional information about it",
		Before: func(ctx *cli.Context) error {
			if err := clicontext.LoadUserConfigQuietly(ctx); err != nil {
				return err
			}
			config.ParseFlagsNode(ctx)
			nodeOptions := node.GetOptions()

			tc := tequilapi_client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort)
			cmd = &command{tequilapi: tc}

			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "proposals",
				Usage: "List all possible proposals to which you can connect",
				Flags: []cli.Flag{&flagCountry, &flagLocationType},
				Action: func(ctx *cli.Context) error {
					cmd.proposals(ctx)
					return nil
				},
			},
		},
	}
}

type command struct {
	tequilapi *tequilapi_client.Client
}

func (c *command) proposals(ctx *cli.Context) {
	locationType := ctx.String(flagLocationType.Name)
	locationCountry := ctx.String(flagCountry.Name)
	if locationCountry != "" && len(locationCountry) != 2 {
		clio.Warn("Country code must be in ISO 3166-1 alpha-2 format. Example: 'UK', 'US'")
		return
	}

	proposals, err := c.tequilapi.ProposalsByLocationAndService(serviceWireguard, locationType, locationCountry)
	if err != nil {
		clio.Warn("Failed to fetch proposal list")
		return
	}

	if len(proposals) == 0 {
		clio.Info("No proposals found")
		return
	}

	clio.Info("Found proposals:")
	for _, p := range proposals {
		printProposal(&p)
	}
}
