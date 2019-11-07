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
	// VendorIDFlag identifies 3rd party vendor (distributor) of Mysterium node
	VendorIDFlag = cli.StringFlag{
		Name: "vendor.id",
		Usage: "Marks vendor (distributor) of the node for collecting statistics. " +
			"3rd party vendors may use their own identifier here.",
	}
)
