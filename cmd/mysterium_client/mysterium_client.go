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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cihub/seelog"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/cli"
	"github.com/mysterium/node/cmd/commands/run"
	"github.com/mysterium/node/core/node"
	_ "github.com/mysterium/node/logconfig"
	"github.com/mysterium/node/metadata"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
)

func main() {
	defer seelog.Flush()
	options, err := run.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	versionSummary := metadata.VersionAsSummary(metadata.LicenseCopyright(
		"run program with '--license.warranty' option",
		"run program with '--license.conditions' option",
	))

	if options.Version {
		fmt.Println(versionSummary)
	} else if options.LicenseWarranty {
		fmt.Println(metadata.LicenseWarranty)
	} else if options.LicenseConditions {
		fmt.Println(metadata.LicenseConditions)
	} else if options.CLI {
		runCLI(options.NodeOptions)
	} else {
		fmt.Println(versionSummary)
		fmt.Println()

		runCMD(options.NodeOptions)
	}
}

func runCLI(options node.NodeOptions) {
	cmdCli := cli.NewCommand(
		filepath.Join(options.Directories.Data, ".cli_history"),
		tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
	)
	stop := cmd.HardKiller(cmdCli.Kill)
	cmd.RegisterSignalCallback(stop)
	if err := cmdCli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCMD(options node.NodeOptions) {
	cmdRun := run.NewCommand(options)
	stop := cmd.SoftKiller(cmdRun.Kill)
	cmd.RegisterSignalCallback(stop)

	if err := cmdRun.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
