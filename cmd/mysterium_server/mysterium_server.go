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
	"github.com/mysterium/node/cmd/commands/server"
	_ "github.com/mysterium/node/logconfig"
	"github.com/mysterium/node/params"
	"os"
)

func main() {
	options, err := server.ParseArguments(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if options.Version {
		printVersion()
	} else if options.LicenseWarranty {
		fmt.Print(params.Warranty)
	} else if options.LicenseConditions {
		fmt.Print(params.Conditions)
	} else {
		printVersion()
		runCMD(options)
	}
}

func printVersion() {
	fmt.Println("Mysterium server")
	fmt.Println("Version:", params.VersionAsString())
	fmt.Println("Build info:", params.BuildAsString())
	fmt.Println("Copyright:", params.GetStartupLicense(
		"run program with '-warranty' option",
		"run program with '-conditions' option",
	))
}

func runCMD(options server.CommandOptions) {
	serverCommand := server.NewCommand(options)

	if err := serverCommand.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.StopOnInterruptsConditional(cmd.NewApplicationStopper(serverCommand.Kill), serverCommand.WaitUnregister)

	if err := serverCommand.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
