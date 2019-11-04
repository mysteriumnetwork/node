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

package license

import (
	"fmt"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/metadata"
	"gopkg.in/urfave/cli.v1"
)

// NewCommand function creates license command
func NewCommand(licenseCopyright string) *cli.Command {
	return &cli.Command{
		Name:      "license",
		Usage:     "Show license",
		ArgsUsage: " ",
		Flags:     []cli.Flag{config.LicenseWarrantyFlag, config.LicenseConditionsFlag},
		Action: func(ctx *cli.Context) error {
			if ctx.IsSet(config.LicenseWarrantyFlag.Name) {
				_, err := fmt.Fprintln(ctx.App.Writer, metadata.LicenseWarranty)
				return err
			}

			if ctx.IsSet(config.LicenseConditionsFlag.Name) {
				_, err := fmt.Fprintln(ctx.App.Writer, metadata.LicenseConditions)
				return err
			}

			_, err := fmt.Fprintln(ctx.App.Writer, licenseCopyright)
			return err
		},
	}
}
