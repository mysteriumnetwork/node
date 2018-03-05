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
