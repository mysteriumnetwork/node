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

package server

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

	Identity   string
	Passphrase string

	LocationCountry  string
	LocationDatabase string

	DiscoveryAPIAddress string
	BrokerAddress       string

	Warranty   bool
	Conditions bool

	IpifyUrl string

	Protocol    string
	OpenvpnPort int
}

const defaultLocationDatabase = "GeoLite2-Country.mmdb"

// TODO: rename to brokerAddress
var natsServerIP string

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
		&options.Protocol,
		"openvpn.proto",
		"udp",
		"Protocol to use. Options: { udp, tcp }",
	)

	flags.IntVar(
		&options.OpenvpnPort,
		"openvpn.port",
		1194,
		"Openvpn port to use. Default 1194",
	)

	flags.StringVar(
		&options.Identity,
		"identity",
		"",
		"Keystore's identity used to provide service. If not given identity will be created automatically",
	)
	flags.StringVar(
		&options.Passphrase,
		"identity.passphrase",
		"",
		"Used to unlock keystore's identity",
	)

	flags.StringVar(
		&options.LocationDatabase,
		"location.database",
		defaultLocationDatabase,
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
	)
	flags.StringVar(
		&options.LocationCountry,
		"location.country",
		"",
		"Service location country. If not given country is autodetected",
	)

	flags.StringVar(
		&options.DiscoveryAPIAddress,
		"discovery-address",
		cmd.MysteriumAPIURL,
		"Address (URL form) of discovery service",
	)
	flags.StringVar(
		&options.BrokerAddress,
		"broker-address",
		natsServerIP,
		"Address (IP or domain name) of message broker",
	)

	flags.BoolVar(
		&options.Warranty,
		"license.warranty",
		false,
		"Show warranty",
	)

	flags.BoolVar(
		&options.Conditions,
		"license.conditions",
		false,
		"Show conditions",
	)

	flags.StringVar(
		&options.IpifyUrl,
		"ipify-url",
		"https://api.ipify.org/",
		"Address (URL form) of ipify service",
	)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
