/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package config

import "github.com/urfave/cli/v2"

var (
	// FlagUIFeatures toggle NodeUI features
	FlagUIFeatures = cli.StringFlag{
		Name:  "ui.features",
		Usage: "Enable NodeUI features. Multiple features are joined by comma (e.g feature1,feature2,...)",
		Value: "",
	}
	// FlagUIEnable enables built-in web UI for node.
	FlagUIEnable = cli.BoolFlag{
		Name:  "ui.enable",
		Usage: "Enables the Web UI",
		Value: true,
	}
	// FlagUIAddress IP address of interface to listen for incoming connections.
	FlagUIAddress = cli.StringFlag{
		Name:  "ui.address",
		Usage: "IP address to bind Web UI to. Address can be comma delimited: '192.168.1.10,192.168.1.20'. (default - 127.0.0.1 and local LAN IP)",
		Value: "",
	}
	// FlagUIPort runs web UI on the specified port.
	FlagUIPort = cli.IntFlag{
		Name:  "ui.port",
		Usage: "The port to run Web UI on",
		Value: 4449,
	}
)

// RegisterFlagsUI register Node UI flags to the list
func RegisterFlagsUI(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagUIFeatures,
		&FlagUIEnable,
		&FlagUIAddress,
		&FlagUIPort,
	)
}

// ParseFlagsUI parse Node UI flags
func ParseFlagsUI(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagUIFeatures)
	Current.ParseBoolFlag(ctx, FlagUIEnable)
	Current.ParseStringFlag(ctx, FlagUIAddress)
	Current.ParseIntFlag(ctx, FlagUIPort)
}
