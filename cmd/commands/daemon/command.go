/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package daemon

import (
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/urfave/cli"
)

// NewCommand function creates run command
func NewCommand() *cli.Command {
	var di cmd.Dependencies

	return &cli.Command{
		Name:      "daemon",
		Usage:     "Starts Mysterium Tequilapi service",
		ArgsUsage: " ",
		Action: func(ctx *cli.Context) error {
			if err := di.Bootstrap(cmd.ParseFlagsNode(ctx)); err != nil {
				return err
			}

			cmd.RegisterSignalCallback(utils.SoftKiller(di.Node.Kill))

			if err := di.Node.Start(); err != nil {
				return err
			}

			return di.Node.Wait()
		},
		After: func(ctx *cli.Context) error {
			err := di.Node.Kill()
			di.Shutdown()
			return err
		},
	}
}
