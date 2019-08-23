/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

// Package urfavecli/cliflags is an adapter to load configuration from urfave/cli.v1 flags

package cliflags

import (
	"github.com/mysteriumnetwork/node/config"
	"gopkg.in/urfave/cli.v1"
)

// SetStringIfPresent helper to register string CLI flag value from urfave/cli.Context
func SetStringIfPresent(cfg *config.Config, name string, ctx *cli.Context) {
	if ctx.IsSet(name) {
		cfg.SetCLI(name, ctx.String(name))
	}
}

// SetIntIfPresent helper to register int CLI flag value from urfave/cli.Context
func SetIntIfPresent(cfg *config.Config, name string, ctx *cli.Context) {
	if ctx.IsSet(name) {
		cfg.SetCLI(name, ctx.Int(name))
	}
}

// SetBoolIfPresent helper to register bool CLI flag value from urfave/cli.Context
func SetBoolIfPresent(cfg *config.Config, name string, ctx *cli.Context) {
	if ctx.IsSet(name) {
		cfg.SetCLI(name, ctx.Bool(name))
	}
}
