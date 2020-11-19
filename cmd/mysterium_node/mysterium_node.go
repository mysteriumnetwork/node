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
	"path/filepath"
	"sync"

	command_cli "github.com/mysteriumnetwork/node/cmd/commands/cli"
	"github.com/mysteriumnetwork/node/cmd/commands/daemon"
	"github.com/mysteriumnetwork/node/cmd/commands/license"
	"github.com/mysteriumnetwork/node/cmd/commands/reset"
	"github.com/mysteriumnetwork/node/cmd/commands/service"
	"github.com/mysteriumnetwork/node/cmd/commands/version"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
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
	resetCommand   = reset.NewCommand()
)

func main() {
	logconfig.Bootstrap()
	app, err := NewCommand()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create command: ")
		os.Exit(1)
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute command: ")
		os.Exit(1)
	}
}

// NewCommand function creates application master command
func NewCommand() (*cli.App, error) {
	cli.VersionPrinter = func(ctx *cli.Context) {
		versionCommand.Run(ctx)
	}

	app, err := newApp()
	if err != nil {
		return nil, err
	}

	app.Usage = "VPN server and client for Mysterium Network https://mysterium.network/"
	app.Authors = []*cli.Author{
		{Name: `The "MysteriumNetwork/node" Authors`, Email: "mysterium-dev@mysterium.network"},
	}
	app.Version = metadata.VersionAsString()
	app.Copyright = licenseCopyright
	app.Before = readyOnceFunc()

	app.Commands = []*cli.Command{
		versionCommand,
		licenseCommand,
		serviceCommand,
		daemonCommand,
		cliCommand,
		resetCommand,
	}

	return app, nil
}

func newApp() (*cli.App, error) {
	app := cli.NewApp()
	return app, config.RegisterFlagsNode(&app.Flags)
}

// uiCommands is a map which consists of all
// commands are used directly by a user.
var uiCommands = map[string]struct{}{
	command_cli.CliCommandName: {},
}

// readyOnceFunc returns a func which is only run once and can
// be executed to preconfigure global settings used in the app.
func readyOnceFunc() cli.BeforeFunc {
	var once sync.Once
	return func(ctx *cli.Context) error {
		once.Do(func() {
			cmd := ctx.Args().First()
			if _, ok := uiCommands[cmd]; !ok {
				// If the command is not meant for user
				// interaction, skip.
				return
			}

			logDir := ctx.String(config.FlagLogDir.Name)
			if logDir == "" {
				// Dont configure UI logger if we have
				// no log dir.
				return
			}

			level, err := zerolog.ParseLevel(ctx.String(config.FlagLogLevel.Name))
			if err != nil {
				level = zerolog.DebugLevel
			}

			err = logconfig.ConfigureUI(&logconfig.LogOptions{
				LogLevel: level,
				LogHTTP:  false,
				Filepath: filepath.Join(logDir, cmd),
			})
			if err != nil {
				log.Error().Err(err).Msg("configuring logger for ui")
			}
		})
		return nil
	}
}
