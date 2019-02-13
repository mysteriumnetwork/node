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
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/urfave/cli"
)

const (
	dataDirFlag    = "data-dir"
	configDirFlag  = "config-dir"
	runtimeDirFlag = "runtime-dir"
)

// RegisterFlagsDirectory function register directory flags to flag list
func RegisterFlagsDirectory(flags *[]cli.Flag) error {
	userHomeDir, err := homedir.Dir()
	if err != nil {
		return err
	}

	currentDir, err := getExecutableDir()
	if err != nil {
		return err
	}

	*flags = append(
		*flags,
		cli.StringFlag{
			Name:  dataDirFlag,
			Usage: "Data directory containing keystore & other persistent files",
			Value: filepath.Join(userHomeDir, ".mysterium"),
		},
		cli.StringFlag{
			Name:  configDirFlag,
			Usage: "Configs directory containing all configuration, script and helper files",
			Value: filepath.Join(currentDir, "config"),
		},
		cli.StringFlag{
			Name:  runtimeDirFlag,
			Usage: "Runtime writable directory for temp files",
			Value: currentDir,
		},
	)
	return nil
}

// ParseFlagsDirectory function fills in directory options from CLI context
func ParseFlagsDirectory(ctx *cli.Context) node.OptionsDirectory {
	dataDir := ctx.GlobalString(dataDirFlag)
	return node.OptionsDirectory{
		dataDir,
		filepath.Join(dataDir, "db"),
		filepath.Join(dataDir, "keystore"),
		ctx.GlobalString(configDirFlag),
		ctx.GlobalString(runtimeDirFlag),
	}
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Dir(executable), nil
}
