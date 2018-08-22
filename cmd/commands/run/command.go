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

package run

import (
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/core/node"
	"github.com/mysterium/node/utils"
	"github.com/urfave/cli"
)

// NewCommand function creates run command
func NewCommand(options node.NodeOptions) *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Runs Mysterium node",
		ArgsUsage: " ",
		Action: func(ctx *cli.Context) error {
			nodeInstance := node.NewNode(options)
			cmd.RegisterSignalCallback(utils.SoftKiller(nodeInstance.Kill))

			if err := nodeInstance.Start(); err != nil {
				return err
			}

			return nodeInstance.Wait()
		},
	}
}
