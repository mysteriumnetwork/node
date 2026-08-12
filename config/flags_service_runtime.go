/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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
	// FlagRuntimeRegistryAddress is the base URL of the corporate registry that
	// lists the runtime workloads this node may install. It has no default:
	// without it, runtime services cannot be created at all, which is the safe
	// state for a node that has no approved workload list. It must be https,
	// because the listing decides what this node executes.
	FlagRuntimeRegistryAddress = cli.StringFlag{
		Name:  "runtime.registry.address",
		Usage: "Base https URL of the runtime service registry listing the workloads this node may create (GET <address>/v1/service)",
	}
)

// RegisterFlagsServiceRuntime registers runtime service flags to the flag list.
func RegisterFlagsServiceRuntime(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagRuntimeRegistryAddress,
	)
}

// ParseFlagsServiceRuntime parses CLI flags and registers values to configuration.
func ParseFlagsServiceRuntime(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagRuntimeRegistryAddress)
}
