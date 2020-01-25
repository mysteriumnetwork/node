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

package config

import (
	"github.com/urfave/cli/v2"
)

var (
	// FlagSOCKS5Port port of listen port.
	FlagSOCKS5Port = cli.IntFlag{
		Name:  "socks5.port",
		Usage: "Port of listen (e.g. 8080)",
		Value: 8080,
	}
)

// RegisterFlagsServiceSOCK5 function register SOCKS5 flags to flag list
func RegisterFlagsServiceSOCK5(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagSOCKS5Port,
	)
}

// ParseFlagsServiceSOCKS5 parses CLI flags and registers value to configuration
func ParseFlagsServiceSOCKS5(ctx *cli.Context) {
	Current.ParseIntFlag(ctx, FlagSOCKS5Port)
}
