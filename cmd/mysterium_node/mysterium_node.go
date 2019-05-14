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
	"os"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/cmd"
	command_cli "github.com/mysteriumnetwork/node/cmd/commands/cli"
	"github.com/mysteriumnetwork/node/cmd/commands/daemon"
	"github.com/mysteriumnetwork/node/cmd/commands/license"
	"github.com/mysteriumnetwork/node/cmd/commands/service"
	"github.com/mysteriumnetwork/node/cmd/commands/version"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli"
)

var (
	licenseCopyright = metadata.LicenseCopyright(
		"run command 'license --warranty'",
		"run command 'license --conditions'",
	)
	versionSummary = metadata.VersionAsSummary(licenseCopyright)
	daemonCommand  = daemon.NewCommand()
	versionCommand = version.NewCommand(versionSummary)
	licenseCommand = license.NewCommand(licenseCopyright)
	serviceCommand = service.NewCommand(licenseCommand.Name)
	cliCommand     = command_cli.NewCommand()
)

func main() {
	app, err := NewCommand()
	if err != nil {
		log.Error("Failed to create command: ", err)
		log.Flush()
		os.Exit(1)
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Error("Failed to execute command: ", err)
		log.Flush()
		os.Exit(1)
	}
}

// NewCommand function creates application master command
func NewCommand() (*cli.App, error) {
	cli.VersionPrinter = func(ctx *cli.Context) {
		versionCommand.Run(ctx)
	}

	app := cli.NewApp()
	app.Usage = "VPN server and client for Mysterium Network https://mysterium.network/"
	app.Authors = []cli.Author{
		{`The "MysteriumNetwork/node" Authors`, "mysterium-dev@mysterium.network"},
	}
	app.Version = metadata.VersionAsString()
	app.Copyright = licenseCopyright
	logconfig.RegisterFlags(&app.Flags)
	if err := cmd.RegisterFlagsNode(&app.Flags); err != nil {
		return nil, err
	}
	app.Commands = []cli.Command{
		*versionCommand,
		*licenseCommand,
		*serviceCommand,
		*daemonCommand,
		*cliCommand,
	}

	return app, nil
}
