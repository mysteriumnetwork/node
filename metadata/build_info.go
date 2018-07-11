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

// Package metadata contains build information of executable usually provided by
// automated build systems like Travis. Default values are populated if not overriden by build system
package metadata

import "fmt"

var (
	// BuildCommit comes from BUILD_COMMIT env variable (set via linker flags)
	BuildCommit = ""
	// BuildBranch comes from BUILD_BRANCH env variable (set via linker flags)
	BuildBranch = "<unknown>"
	// BuildNumber comes from BUILD_NUMBER env variable (set via linker flags)
	BuildNumber = "dev-build"
)

// BuildAsString returns all defined build constants as single string
func BuildAsString() string {
	return FormatString(BuildCommit, BuildBranch, BuildNumber)
}

// FormatString formats build info to string with given build data
func FormatString(commit, branch, buildNumber string) string {
	return fmt.Sprintf("Branch: %s. Build id: %s. Commit: %s", branch, buildNumber, commit)
}
