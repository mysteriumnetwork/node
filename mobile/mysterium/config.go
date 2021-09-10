/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
)

const userConfigFilename = "user-config.toml"

func loadUserConfig(dataDir string) error {
	if err := createDirIfNotExists(dataDir); err != nil {
		return err
	}

	userConfigPath := dataDir + "/" + userConfigFilename
	if err := createFileIfNotExists(userConfigPath); err != nil {
		return err
	}
	if err := config.Current.LoadUserConfig(userConfigPath); err != nil {
		return err
	}

	return nil
}

func createDirIfNotExists(dir string) error {
	err := dirExists(dir)
	if os.IsNotExist(err) {
		log.Info().Msg("Directory does not exist, creating a new one: " + dir)
		return os.MkdirAll(dir, 0700)
	}
	return err
}

func createFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Info().Msg("Config file does not exist, attempting to create: " + filePath)
		_, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create config file %w", err)
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
		return fmt.Errorf("directory expected: %s", dir)
	}
	return nil
}

func setUserConfig(key, value string) error {
	config.Current.SetUser(key, value)
	return config.Current.SaveUserConfig()
}
