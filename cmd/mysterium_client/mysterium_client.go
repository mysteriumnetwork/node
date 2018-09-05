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

	"flag"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/cli"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/utils"
)

func main() {
	options, err := parseArguments(os.Args)
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

type commandOptions struct {
	CLI               bool
	Version           bool
	LicenseWarranty   bool
	LicenseConditions bool

	NodeOptions node.Options
}

func parseArguments(args []string) (options commandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)

	err = cmd.ParseDirectoryArguments(flags, &options.NodeOptions.Directories)
	if err != nil {
		return
	}

	flags.StringVar(
		&options.NodeOptions.Openvpn.Binary,
		"openvpn.binary",
		"openvpn", //search in $PATH by default,
		"openvpn binary to use for Open VPN connections",
	)

	flags.StringVar(
		&options.NodeOptions.TequilapiAddress,
		"tequilapi.address",
		"127.0.0.1",
		"IP address of interface to listen for incoming connections",
	)
	flags.IntVar(
		&options.NodeOptions.TequilapiPort,
		"tequilapi.port",
		4050,
		"Port for listening incoming api requests",
	)

	flags.BoolVar(
		&options.CLI,
		"cli",
		false,
		"Run an interactive CLI based Mysterium UI",
	)
	flags.BoolVar(
		&options.Version,
		"version",
		false,
		"Show version",
	)
	flags.BoolVar(
		&options.LicenseWarranty,
		"license.warranty",
		false,
		"Show warranty",
	)
	flags.BoolVar(
		&options.LicenseConditions,
		"license.conditions",
		false,
		"Show conditions",
	)

	flags.StringVar(
		&options.NodeOptions.Location.IpifyUrl,
		"ipify-url",
		"https://api.ipify.org/",
		"Address (URL form) of ipify service",
	)
	flags.StringVar(
		&options.NodeOptions.Location.Database,
		"location.database",
		"GeoLite2-Country.mmdb",
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
	)

	cmd.ParseNetworkArguments(flags, &options.NodeOptions.NetworkOptions)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}

func runCLI(options node.Options) {
	cmdCli := cli.NewCommand(
		filepath.Join(options.Directories.Data, ".cli_history"),
		tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
	)

	cmd.RegisterSignalCallback(utils.HardKiller(cmdCli.Kill))

	if err := cmdCli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCMD(options node.Options) {
	var di cmd.Dependencies
	if err := di.Bootstrap(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.RegisterSignalCallback(utils.SoftKiller(di.Node.Kill))

	if err := di.Node.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := di.Node.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
