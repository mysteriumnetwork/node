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

package cmd

import (
	"errors"
	"flag"
	log "github.com/cihub/seelog"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

// DirectoryOptions describes data structure holding directories as parameters
type DirectoryOptions struct {
	// Runtime directory for various temp file - usually current working dir
	Runtime string
	// Config directory stores all data needed for runtime (db scripts etc.)
	Config string
	// Data directory stores persistent data like keystore, cli history, etc.
	Data string
}

// ParseFromCmdArgs function takes directory options and fills in values from FlagSet structure
func ParseFromCmdArgs(flags *flag.FlagSet, options *DirectoryOptions) error {

	userHomeDir, err := homedir.Dir()
	if err != nil {
		return err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	flags.StringVar(
		&options.Data,
		"data-dir",
		filepath.Join(userHomeDir, ".mysterium"),
		"Data directory containing keystore & other persistent files",
	)
	flags.StringVar(
		&options.Config,
		"config-dir",
		filepath.Join(currentDir, "config"),
		"Configs directory containing all configuration, script and helper files",
	)
	flags.StringVar(
		&options.Runtime,
		"runtime-dir",
		currentDir,
		"Runtime writable directory for temp files",
	)
	return nil
}

// Check checks that configured dirs exist (which should contain info) and runtime dirs are created (if not exist)
func (options *DirectoryOptions) Check() error {
	err := ensureDirExists(options.Config)
	if err != nil {
		return err
	}

	err = ensureOrCreateDir(options.Runtime)
	if err != nil {
		return err
	}

	return ensureOrCreateDir(options.Data)
}

func ensureOrCreateDir(dir string) error {
	err := ensureDirExists(dir)
	if os.IsNotExist(err) {
		log.Info("[Directory config checker] ", "Directory: ", dir, " does not exit. Creating new one")
		return os.MkdirAll(dir, 0600)
	}
	return err
}

func ensureDirExists(dir string) error {
	fileStat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if isDir := fileStat.IsDir(); !isDir {
		return errors.New("directory expected")
	}
	return nil
}
