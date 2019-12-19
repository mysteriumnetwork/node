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
	"path/filepath"

	"gopkg.in/urfave/cli.v1"
)

var (
	// FlagConfigDir directory containing all configuration, script and helper files.
	FlagConfigDir = cli.StringFlag{
		Name:  "config-dir",
		Usage: "Configs directory containing all configuration, script and helper files",
	}
	// FlagDataDir data directory for keystore and other persistent files.
	FlagDataDir = cli.StringFlag{
		Name:  "data-dir",
		Usage: "Data directory containing keystore & other persistent files",
	}
	// FlagLogDir is a directory for storing log files.
	FlagLogDir = cli.StringFlag{
		Name:  "log-dir",
		Usage: "Log directory for storing log files",
	}
	// FlagRuntimeDir runtime writable directory for temporary files.
	FlagRuntimeDir = cli.StringFlag{
		Name:  "runtime-dir",
		Usage: "Runtime writable directory for temp files",
	}
)

// RegisterFlagsDirectory function register directory flags to flag list
func RegisterFlagsDirectory(flags *[]cli.Flag) error {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	currentDir, err := getExecutableDir()
	if err != nil {
		return err
	}

	FlagConfigDir.Value = filepath.Join(currentDir, "config")
	FlagDataDir.Value = filepath.Join(userHomeDir, ".mysterium")
	FlagLogDir.Value = FlagDataDir.Value
	FlagRuntimeDir.Value = currentDir

	*flags = append(*flags,
		FlagConfigDir,
		FlagDataDir,
		FlagLogDir,
		FlagRuntimeDir,
	)
	return nil
}

// ParseFlagsDirectory function fills in directory options from CLI context
func ParseFlagsDirectory(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagLogDir)
	Current.ParseStringFlag(ctx, FlagDataDir)
	Current.ParseStringFlag(ctx, FlagConfigDir)
	Current.ParseStringFlag(ctx, FlagRuntimeDir)
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Dir(executable), nil
}
