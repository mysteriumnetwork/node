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

// Package version contains build information of executable usually provided by
// automated build systems like Travis. Default values are populated if not overriden by build system
package version

import "fmt"

// Info stores build details
type Info struct {
	Commit      string
	Branch      string
	BuildNumber string
}

// gitCommit comes from COMMIT env variable
var gitCommit = "<unknown>"

// gitBranch comes from BRANCH env variable - if it's github release, this variable will contain release tag name
var gitBranch = "<unknown>"

// buildNumber comes from TRAVIS_JOB_NUMBER env variable
var buildNumber = "dev-build"

// AsString returns all defined build constants as single string
func AsString() string {
	return fmt.Sprintf("Branch: %s. Build id: %s. Commit: %s", gitBranch, buildNumber, gitCommit)
}

// GetInfo returns build details.
func GetInfo() *Info {
	return &Info{
		Commit:      gitCommit,
		Branch:      gitBranch,
		BuildNumber: buildNumber,
	}
}
