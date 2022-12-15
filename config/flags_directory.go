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

	"github.com/urfave/cli/v2"
)

var (
	// FlagConfigDir directory containing all configuration files.
	FlagConfigDir = cli.StringFlag{
		Name:  "config-dir",
		Usage: "Config directory containing all configuration files",
	}
	// FlagDataDir data directory for keystore and other persistent files.
	FlagDataDir = cli.StringFlag{
		Name:  "data-dir",
		Usage: "Data directory containing keystore & other persistent files",
	}
	// FlagNodeUIDir directory containing downloaded nodeUI releases
	FlagNodeUIDir = cli.StringFlag{
		Name:  "node-ui-dir",
		Usage: "Directory containing downloaded nodeUI releases",
	}
	// FlagLogDir is a directory for storing log files.
	FlagLogDir = cli.StringFlag{
		Name:  "log-dir",
		Usage: "Log directory for storing log files. data-dir/logs is used if not specified.",
	}
	// FlagRuntimeDir runtime writable directory for temporary files.
	FlagRuntimeDir = cli.StringFlag{
		Name:  "runtime-dir",
		Usage: "Runtime writable directory for temp files",
	}
	// FlagScriptDir directory containing script and helper files.
	FlagScriptDir = cli.StringFlag{
		Name:  "script-dir",
		Usage: "Script directory containing all script and helper files",
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

	FlagDataDir.Value = filepath.Join(userHomeDir, ".mysterium")
	FlagConfigDir.Value = FlagDataDir.Value
	FlagLogDir.Value = filepath.Join(FlagDataDir.Value, "logs")
	FlagRuntimeDir.Value = currentDir
	FlagScriptDir.Value = filepath.Join(currentDir, "config")
	FlagNodeUIDir.Value = filepath.Join(FlagDataDir.Value, "nodeui")

	*flags = append(*flags,
		&FlagConfigDir,
		&FlagDataDir,
		&FlagLogDir,
		&FlagRuntimeDir,
		&FlagScriptDir,
		&FlagNodeUIDir,
	)
	return nil
}

// ParseFlagsDirectory function fills in directory options from CLI context
func ParseFlagsDirectory(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagConfigDir)
	Current.ParseStringFlag(ctx, FlagDataDir)
	Current.ParseStringFlag(ctx, FlagLogDir)
	Current.ParseStringFlag(ctx, FlagRuntimeDir)
	Current.ParseStringFlag(ctx, FlagScriptDir)
	Current.ParseStringFlag(ctx, FlagNodeUIDir)
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Dir(executable), nil
}
