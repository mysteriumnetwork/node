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
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/cli"
	"github.com/mysterium/node/cmd/commands/client"
	"github.com/mysterium/node/cmd/license"
	_ "github.com/mysterium/node/logconfig"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"os"
	"path/filepath"
)

func main() {
	options, err := client.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if options.CLI {
		runCLI(options)
	} else if options.Warranty {
		fmt.Print(license.Warranty)
	} else if options.Conditions {
		fmt.Print(license.Conditions)
	} else {
		runCMD(options)
	}
}

func runCLI(options client.CommandOptions) {
	cmdCli := cli.NewCommand(
		filepath.Join(options.DirectoryData, ".cli_history"),
		tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
	)
	stop := cmd.NewApplicationStopper(cmdCli.Kill)
	cmd.StopOnInterrupts(stop)
	if err := cmdCli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCMD(options client.CommandOptions) {
	cmdRun := client.NewCommand(options)
	stop := cmd.NewApplicationStopper(cmdRun.Kill)
	cmd.StopOnInterrupts(stop)

	if err := cmdRun.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmdRun.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
