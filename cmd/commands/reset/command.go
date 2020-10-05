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

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/urfave/cli/v2"
)

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
		Name:      "reset",
		Usage:     "Resets Mysterium Node to defaults",
		ArgsUsage: " ",
		Flags:     []cli.Flag{&flagResetTequilapiAuth},
		Action: func(ctx *cli.Context) error {
			config.ParseFlagsNode(ctx)
			nodeOptions := node.GetOptions()

			cmd, err := newResetCommand(ctx.App.Writer, nodeOptions.Directories)
			if err != nil {
				return err
			}

			return cmd.Run(ctx)
		},
	}
}

// newResetCommand creates instance of reset command.
func newResetCommand(
	writer io.Writer,
	dirOptions node.OptionsDirectory,
) (*resetCommand, error) {
	fmt.Println(dirOptions.Storage)

	if err := dirOptions.Check(); err != nil {
		return nil, err
	}

	localStorage, err := boltdb.NewStorage(dirOptions.Storage)
	if err != nil {
		return nil, err
	}

	return &resetCommand{
		writer:  writer,
		storage: localStorage,
	}, nil
}

// resetCommand represent entrypoint for reset command with top level components.
type resetCommand struct {
	writer  io.Writer
	storage *boltdb.Bolt
}

// Run runs a command.
func (rc *resetCommand) Run(ctx *cli.Context) error {
	if ctx.Bool(flagResetTequilapiAuth.Name) {
		if err := rc.resetTequilapi(); err != nil {
			return err
		}
	}

	return nil
}

func (rc *resetCommand) resetTequilapi() error {
	err := auth.NewCredentials(config.FlagTequilapiUsername.Value, config.FlagTequilapiPassword.Value, rc.storage).Set()
	if err != nil {
		return fmt.Errorf("error changing Tequialpi password: %w", err)
	}

	_, _ = fmt.Fprintf(rc.writer, `Tequilapi "%s" user password changed successfully`, config.FlagTequilapiUsername.Value)
	_, _ = fmt.Fprintln(rc.writer)

	return nil
}
