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

package reset

import (
	"fmt"
	"io"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/urfave/cli/v2"
)

// CommandName for the reset command.
const CommandName = "reset"

// flagResetTequilapiAuth instructs to reset Tequilapi auth.
var flagResetTequilapiAuth = cli.BoolFlag{
	Name: "tequilapi",
	Usage: fmt.Sprintf("Reset Tequilapi auth credentials to values provider in --%s and --%s flags",
		config.FlagTequilapiUsername.Name,
		config.FlagTequilapiPassword.Name,
	),
	Value: true,
}

// NewCommand creates reset command.
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:      CommandName,
		Usage:     "Resets Mysterium Node to defaults",
		ArgsUsage: " ",
		Flags:     []cli.Flag{&flagResetTequilapiAuth},
		Action: func(ctx *cli.Context) error {
			cmd, err := newAction(ctx)
			if err != nil {
				return err
			}

			return cmd.Run(ctx)
		},
	}
}

// newAction creates instance of reset action.
func newAction(ctx *cli.Context) (*resetAction, error) {
	client, err := clio.NewTequilApiClient(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := remote.NewConfig(client)
	if err != nil {
		return nil, err
	}

	return &resetAction{
		writer: ctx.App.Writer,
		cfg:    cfg,
	}, nil
}

// resetAction represent entrypoint for reset command with top level components.
type resetAction struct {
	writer io.Writer
	cfg    *remote.Config
}

// Run runs action tasks.
func (rc *resetAction) Run(ctx *cli.Context) error {
	if ctx.Bool(flagResetTequilapiAuth.Name) {
		err := rc.resetTequilapi()
		if err != nil {
			fmt.Fprintln(rc.writer, err)
		}
		return err
	}

	return nil
}

func (rc *resetAction) resetTequilapi() error {
	err := auth.
		NewCredentialsManager(rc.cfg.GetString(config.FlagDataDir.Name)).
		SetPassword(rc.cfg.GetString(config.FlagTequilapiPassword.Name))
	if err != nil {
		return fmt.Errorf("error changing Tequialpi password: %w", err)
	}

	_, _ = fmt.Fprintf(rc.writer, `Tequilapi "%s" user password changed successfully`, config.FlagTequilapiUsername.Value)
	_, _ = fmt.Fprintln(rc.writer)

	return nil
}
