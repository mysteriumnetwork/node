/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/supervisor/daemon/transport"
	"github.com/mysteriumnetwork/node/supervisor/logconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/supervisor/config"
	"github.com/mysteriumnetwork/node/supervisor/daemon"
	"github.com/mysteriumnetwork/node/supervisor/install"
)

var (
	flagVersion    = flag.Bool("version", false, "Print version")
	flagInstall    = flag.Bool("install", false, "Install or repair myst supervisor")
	flagUninstall  = flag.Bool("uninstall", false, "Uninstall myst supervisor")
	logFilePath    = flag.String("log-path", "", "Supervisor log file path")
	logLevel       = flag.String("log-level", zerolog.InfoLevel.String(), "Logging level")
	flagWinService = flag.Bool("winservice", false, "Run via service manager instead of standalone (windows only).")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println(metadata.VersionAsString())
		os.Exit(0)
	}

	logOpts := logconfig.LogOptions{
		LogLevel: *logLevel,
		Filepath: *logFilePath,
	}
	if err := logconfig.Configure(logOpts); err != nil {
		log.Fatal().Err(err).Msg("Failed to configure logging")
	}

	if *flagInstall {
		path, err := thisPath()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to determine supervisor's path")
		}

		options := install.Options{
			SupervisorPath: path,
		}
		log.Info().Msgf("Installing supervisor with options: %#v", options)
		if err = install.Install(options); err != nil {
			log.Fatal().Err(err).Msg("Failed to install supervisor")
		}
		log.Info().Msg("Supervisor installed")
	} else if *flagUninstall {
		log.Info().Msg("Uninstalling supervisor")
		if err := install.Uninstall(); err != nil {
			log.Fatal().Err(err).Msg("Failed to uninstall supervisor")
		}
		log.Info().Msg("Supervisor uninstalled")
	} else {
		log.Info().Msg("Running myst supervisor daemon")
		supervisor := daemon.New(&config.Config{})
		if err := supervisor.Start(transport.Options{WinService: *flagWinService}); err != nil {
			log.Fatal().Err(err).Msg("Error running supervisor")
		}
	}
}

func thisPath() (string, error) {
	thisExec, err := os.Executable()
	if err != nil {
		return "", err
	}
	thisPath, err := filepath.Abs(thisExec)
	if err != nil {
		return "", err
	}
	return thisPath, nil
}
