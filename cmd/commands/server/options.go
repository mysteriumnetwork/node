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
)

// CommandOptions describes options which are required to start Command
type CommandOptions struct {
	Directories   cmd.DirectoryOptions
	OpenvpnBinary string

	Identity   string
	Passphrase string

	LocationCountry  string
	LocationDatabase string

	Version           bool
	LicenseWarranty   bool
	LicenseConditions bool

	IpifyUrl string

	Protocol    string
	OpenvpnPort int

	AgreedTermsConditions bool
	cmd.NetworkOptions
}

// ParseArguments parses CLI flags and adds to CommandOptions structure
func ParseArguments(args []string) (options CommandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)

	err = cmd.ParseFromCmdArgs(flags, &options.Directories)
	if err != nil {
		return
	}

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
		"GeoLite2-Country.mmdb",
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
	)
	flags.StringVar(
		&options.LocationCountry,
		"location.country",
		"",
		"Service location country. If not given country is autodetected",
	)

	flags.BoolVar(
		&options.AgreedTermsConditions,
		"agreed-terms-and-conditions",
		false,
		"agree with terms & conditions",
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
		&options.IpifyUrl,
		"ipify-url",
		"https://api.ipify.org/",
		"Address (URL form) of ipify service",
	)

	cmd.ParseNetworkOptions(flags, &options.NetworkOptions)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}
