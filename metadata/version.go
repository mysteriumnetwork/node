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

	"github.com/mysteriumnetwork/terms/terms-go"
)

// Version comes from BUILD_VERSION env variable (set via linker flags)
var Version = ""

const versionSummaryFormat = `Mysterium Node
  Version: %s
  Build info: %s

%s
%s`

// VersionAsString returns all defined version constants as single string
func VersionAsString() string {
	if Version != "" {
		return Version
	}

	version := "source"
	if len(BuildCommit) >= 8 {
		version += "." + BuildCommit[:8]
	} else if BuildNumber != "" {
		version += "." + BuildNumber
	}
	return version
}

// VersionAsSummary returns overview of current program's version
func VersionAsSummary(licenseCopyright string) string {
	return fmt.Sprintf(
		versionSummaryFormat,
		VersionAsString(),
		BuildAsString(),
		licenseCopyright,
		string(terms.TermsNodeShort),
	)
}
