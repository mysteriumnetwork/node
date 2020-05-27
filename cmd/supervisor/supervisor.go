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
	"log"
	"os"
	"path/filepath"

	"github.com/mysteriumnetwork/node/supervisor/config"
	"github.com/mysteriumnetwork/node/supervisor/daemon"
	"github.com/mysteriumnetwork/node/supervisor/install"
)

var (
	flagInstall = flag.Bool("install", false, "Install or repair myst supervisor")
)

func main() {
	flag.Parse()

	if *flagInstall {
		log.Println("Installing supervisor")
		path, err := thisPath()
		if err != nil {
			log.Fatalln("Failed to determine supervisor's path:", err)
		}
		err = install.Install(install.Options{
			SupervisorPath: path,
		})
		if err != nil {
			log.Fatalln("Failed to install supervisor:", err)
		}
	} else {
		log.Println("Running myst supervisor daemon")
		supervisor := daemon.New(&config.Config{})
		if err := supervisor.Start(); err != nil {
			log.Fatalln("Error running supervisor:", err)
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
