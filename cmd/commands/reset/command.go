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
	var action *resetAction

	return &cli.Command{
		Name:      "reset",
		Usage:     "Resets Mysterium Node to defaults",
		ArgsUsage: " ",
		Flags:     []cli.Flag{&flagResetTequilapiAuth},
		Action: func(ctx *cli.Context) error {
			action, err := newAction(ctx)
			if err != nil {
				return err
			}

			return action.Run(ctx)
		},
		After: func(ctx *cli.Context) error {
			if action == nil {
				return nil
			}

			return action.Cleanup(ctx)
		},
	}
}

// newAction creates instance of reset action.
func newAction(ctx *cli.Context) (*resetAction, error) {
	config.ParseFlagsNode(ctx)

	nodeOptions := node.GetOptions()
	if err := nodeOptions.Directories.Check(); err != nil {
		return nil, err
	}

	storage, err := boltdb.NewStorage(nodeOptions.Directories.Storage)
	if err != nil {
		return nil, err
	}

	return &resetAction{
		writer:  ctx.App.Writer,
		storage: storage,
	}, nil
}

// resetAction represent entrypoint for reset command with top level components.
type resetAction struct {
	writer  io.Writer
	storage *boltdb.Bolt
}

// Run runs action tasks.
func (rc *resetAction) Run(ctx *cli.Context) error {
	if ctx.Bool(flagResetTequilapiAuth.Name) {
		return rc.resetTequilapi()
	}

	return nil
}

// Cleanup runs action cleanup tasks.
func (rc *resetAction) Cleanup(_ *cli.Context) error {
	return rc.storage.Close()
}

func (rc *resetAction) resetTequilapi() error {
	err := auth.NewCredentials(config.FlagTequilapiUsername.Value, config.FlagTequilapiPassword.Value, rc.storage).Set()
	if err != nil {
		return fmt.Errorf("error changing Tequialpi password: %w", err)
	}

	_, _ = fmt.Fprintf(rc.writer, `Tequilapi "%s" user password changed successfully`, config.FlagTequilapiUsername.Value)
	_, _ = fmt.Fprintln(rc.writer)

	return nil
}
