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
	"log"
	"os"
	"path/filepath"

	"github.com/mysteriumnetwork/node/supervisor/config"
	"github.com/mysteriumnetwork/node/supervisor/daemon"
	"github.com/mysteriumnetwork/node/supervisor/install"
)

var (
	flagInstall     = flag.Bool("install", false, "Install or repair myst supervisor")
	flagMystPath    = flag.String("mystPath", "", "Path to myst executable (required for -install)")
	flagOpenVPNPath = flag.String("openvpnPath", "", "Path to openvpn executable (required for -install)")
)

func ensureInstallFlags() {
	if *flagMystPath == "" || *flagOpenVPNPath == "" {
		fmt.Println("Error: required flags were not set")
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	if *flagInstall {
		ensureInstallFlags()
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
		log.Println("Creating supervisor configuration")
		cfg := config.Config{
			MystPath:    *flagMystPath,
			OpenVPNPath: *flagOpenVPNPath,
		}
		err = cfg.Write()
		if err != nil {
			log.Fatalln("Failed to create supervisor configuration:", err)
		}
	} else {
		log.Println("Running myst supervisor daemon")
		cfg, err := config.Read()
		if err != nil {
			log.Println("Failed to read supervisor configuration:", err)
		}
		supervisor := daemon.New(cfg)
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
