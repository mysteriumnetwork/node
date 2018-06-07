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
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/cmd"
	"github.com/mysterium/node/cmd/commands/server"
	"github.com/mysterium/node/cmd/commands/server/terms_and_conditions"
	"github.com/mysterium/node/cmd/license"
	_ "github.com/mysterium/node/logconfig"
	"github.com/mysterium/node/metadata"
	"os"
)

func main() {
	defer log.Flush()
	options, err := server.ParseArguments(os.Args)
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
	} else {
		fmt.Println(versionSummary)
		fmt.Println()

		runCMD(options)
	}
	startWithOptions(options)
}

func startWithOptions(options server.CommandOptions) {
	if !terms_and_conditions.UserAgreed(options.AgreedTermsConditions) {
		fmt.Print(terms_and_conditions.Text + "\n")
		return
	}
	log.Infof("User agreed with terms and conditions: %v", options.AgreedTermsConditions)

	if options.LicenseWarranty {
		fmt.Print(license.Warranty)
		return
	}
	if options.LicenseConditions {
		fmt.Print(license.Conditions)
		return
	}

	runCMD(options)
}

func runCMD(options server.CommandOptions) {
	serverCommand := server.NewCommand(options)

	if err := serverCommand.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.RegisterSignalCallback(cmd.SoftKiller(serverCommand.Kill))

	if err := serverCommand.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
