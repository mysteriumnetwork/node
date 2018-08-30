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

package cmd

import (
	"github.com/mysterium/node/core/node"
	"github.com/urfave/cli"
)

var (
	tequilapiAddressFlag = cli.StringFlag{
		Name:  "tequilapi.address",
		Usage: "IP address of interface to listen for incoming connections",
		Value: "127.0.0.1",
	}
	tequilapiPortFlag = cli.IntFlag{
		Name:  "tequilapi.port",
		Usage: "Port for listening incoming api requests",
		Value: 4050,
	}

	openvpnBinaryFlag = cli.StringFlag{
		Name:  "openvpn.binary",
		Usage: "openvpn binary to use for Open VPN connections",
		Value: "openvpn",
	}
	ipifyUrlFlag = cli.StringFlag{
		Name:  "ipify-url",
		Usage: "Address (URL form) of ipify service",
		Value: "https://api.ipify.org/",
	}
	locationDatabaseFlag = cli.StringFlag{
		Name:  "location.database",
		Usage: "Service location autodetect database of GeoLite2 format e.g. http://dev.maxmind.com/geoip/geoip2/geolite2/",
		Value: "GeoLite2-Country.mmdb",
	}
)

// RegisterNodeFlags function register node flags to flag list
func RegisterNodeFlags(flags *[]cli.Flag) error {
	if err := RegisterDirectoryFlags(flags); err != nil {
		return err
	}

	*flags = append(*flags, tequilapiAddressFlag, tequilapiPortFlag)

	RegisterNetworkFlags(flags)

	*flags = append(*flags, openvpnBinaryFlag, ipifyUrlFlag, locationDatabaseFlag)

	return nil
}

// ParseNodeFlags function fills in node options from CLI context
func ParseNodeFlags(ctx *cli.Context) node.Options {
	return node.Options{
		ParseDirectoryFlags(ctx),

		ctx.GlobalString(tequilapiAddressFlag.Name),
		ctx.GlobalInt(tequilapiPortFlag.Name),

		ctx.GlobalString(openvpnBinaryFlag.Name),
		ctx.GlobalString(ipifyUrlFlag.Name),
		ctx.GlobalString(locationDatabaseFlag.Name),

		ParseNetworkFlags(ctx),
	}
}
