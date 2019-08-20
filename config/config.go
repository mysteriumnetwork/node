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
	"os"
	"path"

	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

var log = logconfig.NewLogger()

type configLocation struct {
	dir      string
	filePath string
}

var location = configLocation{}

// LoadConfigurationFile loads configuration values from config file in home directory
func LoadConfigurationFile(ctx *cli.Context) error {
	location = resolveLocation(ctx)
	log.Infof("using config location: %+v", location)

	err := createConfigIfNotExists(location)
	if err != nil {
		return err
	}

	configSource, err := altsrc.NewTomlSourceFromFile(location.filePath)
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

func resolveLocation(ctx *cli.Context) configLocation {
	dir := ctx.GlobalString("config-dir")
	return configLocation{
		dir:      dir,
		filePath: path.Join(dir, "config.toml"),
	}
}

func createConfigIfNotExists(location configLocation) error {
	err := createDirIfNotExists(location.dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(location.filePath); os.IsNotExist(err) {
		log.Info("config file does not exist, attempting to create: ", location.filePath)
		_, err := os.Create(location.filePath)
		if err != nil {
			return errors.Wrap(err, "failed to create config file")
		}
	}
	return nil
}

func createDirIfNotExists(dir string) error {
	err := dirExists(dir)
	if os.IsNotExist(err) {
		log.Info("directory does not exist, creating a new one: ", dir)
		return os.MkdirAll(dir, 0700)
	}
	return err
}

func dirExists(dir string) error {
	fileStat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if isDir := fileStat.IsDir(); !isDir {
		return errors.New("directory expected")
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
