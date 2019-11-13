/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package node

import (
	"os"
	"path/filepath"

	"github.com/mysteriumnetwork/node/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// OptionsDirectory describes data structure holding directories as parameters.
type OptionsDirectory struct {
	// Data directory stores persistent data like keystore, cli history, etc.
	Data string
	// Data directory stores database
	Storage string
	// Data directory stores identity keys
	Keystore string
	// Config directory stores all data needed for runtime (db scripts etc.)
	Config string
	// Runtime directory for various temp file - usually current working dir
	Runtime string
}

// GetOptionsDirectory retrieves directory configuration from app configuration.
func GetOptionsDirectory() *OptionsDirectory {
	dataDir := config.GetString(config.FlagDataDir)
	return &OptionsDirectory{
		Data:     dataDir,
		Storage:  filepath.Join(dataDir, "db"),
		Keystore: filepath.Join(dataDir, "keystore"),
		Config:   config.GetString(config.FlagConfigDir),
		Runtime:  config.GetString(config.FlagRuntimeDir),
	}
}

// Check checks that configured dirs exist (which should contain info) and runtime dirs are created (if not exist)
func (options *OptionsDirectory) Check() error {
	err := ensureOrCreateDir(options.Runtime)
	if err != nil {
		return err
	}

	err = ensureOrCreateDir(options.Storage)
	if err != nil {
		return err
	}

	return ensureOrCreateDir(options.Data)
}

func ensureOrCreateDir(dir string) error {
	err := ensureDirExists(dir)
	if os.IsNotExist(err) {
		log.Info().Msg("Directory does not exist, creating a new one: " + dir)
		return os.MkdirAll(dir, 0700)
	}
	return err
}

func ensureDirExists(dir string) error {
	fileStat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if isDir := fileStat.IsDir(); !isDir {
		return errors.Errorf("directory expected: %s", dir)
	}
	return nil
}
