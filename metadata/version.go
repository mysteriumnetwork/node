/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package metadata

import (
	"fmt"
)

const (
	// VersionMajor is version component of the current release
	VersionMajor = 0
	// VersionMinor is version component of the current release
	VersionMinor = 1
	// VersionPatch is version component of the current release
	VersionPatch = 0
)

// VersionAsString returns all defined version constants as single string
func VersionAsString() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}

const versionSummaryFormat = `Mysterium Node
  Version: %s
  Build info: %s
%s`

// VersionAsSummary returns overview of current program's version
func VersionAsSummary(licenseCopyright string) string {
	return fmt.Sprintf(
		versionSummaryFormat,
		VersionAsString(),
		BuildAsString(),
		licenseCopyright,
	)
}
