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

package config

import (
	"path"

	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

var log = logconfig.NewLogger()
var configFilePath string

// LoadConfigurationFile loads configuration values from config file in home directory
func LoadConfigurationFile(ctx *cli.Context) error {
	configDir := ctx.GlobalString("config-dir")
	log.Info("using config directory: ", configDir)

	configFilePath = path.Join(configDir, "config.toml")
	configSource, err := altsrc.NewTomlSourceFromFile(configFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to load config file")
	}
	flags := allFlags(ctx)
	err = altsrc.ApplyInputSourceValues(ctx, configSource, flags)
	if err != nil {
		return errors.Wrap(err, "failed to apply configuration from config file")
	}
	return nil
}

// LoadConfigurationFileQuietly like LoadConfigurationFile, but instead of returning an error,
// it logs it on a `warn` level.
// `error` is specified as a return to adhere to `cli.BeforeFunc` for convenience.
func LoadConfigurationFileQuietly(ctx *cli.Context) error {
	err := LoadConfigurationFile(ctx)
	if err != nil {
		_ = log.Warn(err)
	}
	return nil
}

func allFlags(ctx *cli.Context) []cli.Flag {
	var flags []cli.Flag
	flags = append(flags, ctx.App.Flags...)
	flags = append(flags, ctx.Command.Flags...)
	for _, cmd := range ctx.Command.Subcommands {
		flags = append(flags, cmd.Flags...)
	}
	return flags
}
