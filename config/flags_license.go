/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

import "gopkg.in/urfave/cli.v1"

var (
	// LicenseWarrantyFlag flag allows to print license warranty
	LicenseWarrantyFlag = cli.BoolFlag{
		Name:  "warranty",
		Usage: "Show details of license warranty",
	}
	// LicenseConditionsFlag flag allows to print license conditions
	LicenseConditionsFlag = cli.BoolFlag{
		Name:  "conditions",
		Usage: "Show details of license conditions",
	}
)
