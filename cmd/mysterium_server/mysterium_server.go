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
	"flag"
	"fmt"
	"os"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/metadata"
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
	} else if !options.AgreedTermsConditions {
		fmt.Println(versionSummary)
		fmt.Println()

		fmt.Println("If you agree with these Terms & Conditions, run program again with '--agreed-terms-and-conditions' flag.")
		os.Exit(2)
	} else {
		fmt.Println(versionSummary)
		fmt.Println()

		fmt.Printf("User agreed with terms & conditions: %v\n", options.AgreedTermsConditions)
		runCMD(options.NodeOptions, options.ServiceOptions)
	}
}

type commandOptions struct {
	Version               bool
	LicenseWarranty       bool
	LicenseConditions     bool
	AgreedTermsConditions bool

	NodeOptions    node.Options
	ServiceOptions service.Options
}

func parseArguments(args []string) (options commandOptions, err error) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)

	err = cmd.ParseDirectoryArguments(flags, &options.NodeOptions.Directories)
	if err != nil {
		return
	}

	flags.StringVar(
		&options.NodeOptions.OpenvpnBinary,
		"openvpn.binary",
		"openvpn", //search in $PATH by default,
		"openvpn binary to use for Open VPN connections",
	)
	flags.StringVar(
		&options.ServiceOptions.OpenvpnProtocol,
		"openvpn.proto",
		"udp",
		"Openvpn protocol to use. Options: { udp, tcp }",
	)
	flags.IntVar(
		&options.ServiceOptions.OpenvpnPort,
		"openvpn.port",
		1194,
		"Openvpn port to use. Default 1194",
	)

	flags.StringVar(
		&options.ServiceOptions.Identity,
		"identity",
		"",
		"Keystore's identity used to provide service. If not given identity will be created automatically",
	)
	flags.StringVar(
		&options.ServiceOptions.Passphrase,
		"identity.passphrase",
		"",
		"Used to unlock keystore's identity",
	)

	flags.StringVar(
		&options.NodeOptions.Location.Database,
		"location.database",
		"GeoLite2-Country.mmdb",
		"Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
	)
	flags.StringVar(
		&options.NodeOptions.Location.Country,
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
		&options.NodeOptions.IpifyUrl,
		"ipify-url",
		"https://api.ipify.org/",
		"Address (URL form) of ipify service",
	)

	cmd.ParseNetworkArguments(flags, &options.NodeOptions.NetworkOptions)

	err = flags.Parse(args[1:])
	if err != nil {
		return
	}

	return options, err
}

func runCMD(nodeOptions node.Options, serviceOptions service.Options) {
	serviceManager := service.NewManager(nodeOptions, serviceOptions)

	if err := serviceManager.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd.RegisterSignalCallback(utils.SoftKiller(serviceManager.Kill))

	if err := serviceManager.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
