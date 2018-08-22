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

	"github.com/mysterium/node/cmd"
	command_cli "github.com/mysterium/node/cmd/commands/cli"
	"github.com/mysterium/node/cmd/commands/run"
	"github.com/mysterium/node/cmd/commands/version"
	"github.com/mysterium/node/core/node"
	"github.com/mysterium/node/metadata"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"github.com/mysterium/node/utils"
	"github.com/urfave/cli"
)

func main() {
	err := NewCommand().Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewCommand function creates application master command
func NewCommand() *cli.App {
	app := cli.NewApp()
	app.Usage = "VPN server and client for Mysterium Network https://mysterium.network/"
	app.Authors = []cli.Author{
		{`The "MysteriumNetwork/node" Authors`, "mysterium-dev@mysterium.network"},
	}
	app.Version = metadata.VersionAsString()
	app.Commands = []cli.Command{
		*versionCommand,
	}
	app.Flags = []cli.Flag{
		tequilapiAddressFlag,
		tequilapiPortFlag,

		licenseWarrantyFlag,
		licenseConditionsFlag,

		openvpnBinaryFlag,
		ipifyUrlFlag,
		locationDatabaseFlag,
		cliFlag,
	}
	app.Action = runMain

	cli.VersionPrinter = func(ctx *cli.Context) {
		versionCommand.Run(ctx)
	}

	return app
}

func runMain(ctx *cli.Context) error {
	if options.LicenseWarranty {
		fmt.Println(metadata.LicenseWarranty)
		return nil
	} else if options.LicenseConditions {
		fmt.Println(metadata.LicenseConditions)
		return nil
	} else if options.CLI {
		return runCLI(options.NodeOptions)
	} else {
		fmt.Println(versionSummary)
		fmt.Println()

		return run.NewCommand(options.NodeOptions).Run(ctx)
	}
}

func runCLI(options node.NodeOptions) error {
	cmdCli := command_cli.NewCommand(
		filepath.Join(options.Directories.Data, ".cli_history"),
		tequilapi_client.NewClient(options.TequilapiAddress, options.TequilapiPort),
	)
	cmd.RegisterSignalCallback(utils.HardKiller(cmdCli.Kill))

	return cmdCli.Run()
}

type commandOptions struct {
	CLI               bool
	LicenseWarranty   bool
	LicenseConditions bool

	NodeOptions node.NodeOptions
}

var (
	options commandOptions

	versionSummary = metadata.VersionAsSummary(metadata.LicenseCopyright(
		"command_run program with '--license.warranty' option",
		"command_run program with '--license.conditions' option",
	))
	versionCommand = version.NewCommand(versionSummary)

	tequilapiAddressFlag = cli.StringFlag{
		Name:        "tequilapi.address",
		Usage:       "IP address of interface to listen for incoming connections",
		Destination: &options.NodeOptions.TequilapiAddress,
		Value:       "127.0.0.1",
	}
	tequilapiPortFlag = cli.IntFlag{
		Name:        "tequilapi.port",
		Usage:       "Port for listening incoming api requests",
		Destination: &options.NodeOptions.TequilapiPort,
		Value:       4050,
	}

	licenseWarrantyFlag = cli.BoolFlag{
		Name:        "license.warranty",
		Usage:       "Show warranty",
		Destination: &options.LicenseWarranty,
	}
	licenseConditionsFlag = cli.BoolFlag{
		Name:        "license.conditions",
		Usage:       "Show conditions",
		Destination: &options.LicenseConditions,
	}

	openvpnBinaryFlag = cli.StringFlag{
		Name:        "openvpn.binary",
		Usage:       "openvpn binary to use for Open VPN connections",
		Destination: &options.NodeOptions.OpenvpnBinary,
		Value:       "openvpn",
	}
	ipifyUrlFlag = cli.StringFlag{
		Name:        "ipify-url",
		Usage:       "Address (URL form) of ipify service",
		Destination: &options.NodeOptions.IpifyUrl,
		Value:       "https://api.ipify.org/",
	}
	locationDatabaseFlag = cli.StringFlag{
		Name:        "location.database",
		Usage:       "Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
		Destination: &options.NodeOptions.LocationDatabase,
		Value:       "GeoLite2-Country.mmdb",
	}
	cliFlag = cli.BoolFlag{
		Name:        "cli",
		Usage:       "Run an interactive CLI based Mysterium UI",
		Destination: &options.CLI,
	}
)
