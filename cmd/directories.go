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
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

// GetDataDirectory makes full path to application's data
func GetDataDirectory() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, ".mysterium")
}

// DirectoryOptions describes data structure holding directories as parameters
type DirectoryOptions struct {
	RuntimeDir string
	ConfigDir  string
	DataDir    string
}

// ParseFromCmdArgs function takes directory options and fills in values from FlagSet structure
func ParseFromCmdArgs(flags *flag.FlagSet, options *DirectoryOptions) error {

	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	flags.StringVar(
		&options.DataDir,
		"data-dir",
		filepath.Join(workingDir, ".mysterium"),
		"Data directory containing keystore & other persistent files",
	)
	flags.StringVar(
		&options.ConfigDir,
		"config-dir",
		filepath.Join(workingDir, "config"),
		"Configs directory containing all configuration, script and helper files",
	)
	flags.StringVar(
		&options.RuntimeDir,
		"runtime-dir",
		workingDir,
		"Runtime writable directory for temp files",
	)
	return nil
}

// Check checks that configured dirs exist (which should contain info) and runtime dirs are created (if not exist)
func (options *DirectoryOptions) Check() error {
	err := ensureDirExists(options.ConfigDir)
	if err != nil {
		return err
	}

	err = ensureOrCreateDir(options.RuntimeDir)
	if err != nil {
		return err
	}

	return ensureOrCreateDir(options.DataDir)
}

func ensureOrCreateDir(dir string) error {
	err := ensureDirExists(dir)
	if os.IsNotExist(err) {
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
