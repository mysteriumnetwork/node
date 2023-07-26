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
	// FlagNodeVersion - stores node version to discover a fact of node update
	FlagNodeVersion = cli.StringFlag{
		Name:   "node.version",
		Usage:  "",
		Value:  "",
		Hidden: true,
	}
)

// RegisterFlagNodeVersion register Node version flags to the list
func RegisterFlagNodeVersion(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagNodeVersion,
	)
}
