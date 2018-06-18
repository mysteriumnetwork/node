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

package client

import (
	"flag"
	"github.com/mysterium/node/cmd"
	"path/filepath"
)

// CommandOptions describes options which are required to start Command
type CommandOptions struct {
	DirectoryConfig  string
	DirectoryRuntime string
	DirectoryData    string
	OpenvpnBinary    string

	TequilapiAddress string
	TequilapiPort    int

	CLI               bool
	Version           bool
	LicenseWarranty   bool
	LicenseConditions bool

	DiscoveryAPIAddress string
	BrokerAddress       string
	IpifyUrl            string

	LocationDatabase string
}

// ParseArguments parses CLI flags and adds to CommandOptions structure
func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(
		&options.DirectoryData,
		"data-dir",
		cmd.GetDataDirectory(),
		"Data directory containing keystore & other persistent files",
	)
	flags.StringVar(
		&options.DirectoryConfig,
		"config-dir",
		filepath.Join(cmd.GetDataDirectory(), "config"),
		"Configs directory containing all configuration files",
	)
	flags.StringVar(
		&options.DirectoryRuntime,
		"runtime-dir",
		filepath.Join(cmd.GetDataDirectory(), "run"),
		"Runtime writable directory for temp files",
	)
	flags.StringVar(
		&options.OpenvpnBinary,
		"openvpn.binary",
		"openvpn", //search in $PATH by default,
		"openvpn binary to use for Open VPN connections",
	)

	flags.StringVar(
		&options.TequilapiAddress,
		"tequilapi.address",
		"127.0.0.1",
		"IP address of interface to listen for incoming connections",
	)
	flags.IntVar(
		&options.TequilapiPort,
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
		&options.DiscoveryAPIAddress,
		"discovery-address",
		cmd.MysteriumAPIURL,
		"Address (URL form) of discovery service",
	)

	flags.StringVar(
		&options.IpifyUrl,
		"ipify-url",
		"https://api.ipify.org/",
		"Address (URL form) of ipify service",
	)

	flags.StringVar(
		&options.LocationDatabase,
		"location.database",
		"GeoLite2-Country.mmdb",
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
