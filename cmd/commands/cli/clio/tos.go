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

package clio

import (
	"fmt"

	"github.com/mysteriumnetwork/node/metadata"
)

// PrintTOSError prints TOS together with a given error
// asking user to accept them.
func PrintTOSError(err error) {
	fmt.Println(metadata.VersionAsSummary(metadata.LicenseCopyright(
		"type 'license --warranty'",
		"type 'license --conditions'",
	)))
	fmt.Println()
	Error(err)
	Info("If you agree with these Terms & Conditions, run program again with '--agreed-terms-and-conditions' flag")
}
