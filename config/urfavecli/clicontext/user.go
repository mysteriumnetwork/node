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

// Package urfavecli/clicontext is an adapter to load configuration from urfave/cli.v1 Context

package clicontext

import (
	"os"
	"path"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
)

var log = logconfig.NewLogger()

// LoadUserConfig determines config location from the context
// and makes sure that the config file actually exists, creating it if necessary
func LoadUserConfig(ctx *cli.Context) error {
	configDir, configFilePath := resolveLocation(ctx)
	err := createDirIfNotExists(configDir)
	if err != nil {
		return err
	}
	err = createFileIfNotExists(configFilePath)
	if err != nil {
		return err
	}

	return config.Current.LoadUserConfig(configFilePath)
}

// LoadUserConfigQuietly like LoadUserConfig, but instead of returning an error,
// it logs it on a `warn` level.
// `error` is specified as a return to adhere to `cli.BeforeFunc` for convenience.
func LoadUserConfigQuietly(ctx *cli.Context) error {
	err := LoadUserConfig(ctx)
	if err != nil {
		log.Warn(err)
	}
	return nil
}

func resolveLocation(ctx *cli.Context) (configDir string, configFilePath string) {
	configDir = ctx.GlobalString("config-dir")
	configFilePath = path.Join(configDir, "config.toml")
	return configDir, configFilePath
}

func createDirIfNotExists(dir string) error {
	err := dirExists(dir)
	if os.IsNotExist(err) {
		log.Info("directory does not exist, creating a new one: ", dir)
		return os.MkdirAll(dir, 0700)
	}
	return err
}

func createFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Info("config file does not exist, attempting to create: ", filePath)
		_, err := os.Create(filePath)
		if err != nil {
			return errors.Wrap(err, "failed to create config file")
		}
	}
	return nil
}

func dirExists(dir string) error {
	fileStat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if isDir := fileStat.IsDir(); !isDir {
		return errors.Errorf("directory expected: %s", dir)
	}
	return nil
}
