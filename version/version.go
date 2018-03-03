// Package version contains build information of executable usually provided by
// automated build systems like Travis. Default values are populated if not overriden by build system
package version

import "fmt"

type Info struct {
	Commit      string
	Branch      string
	BuildNumber string
}

// GitCommit comes from TRAVIS_COMMIT env variable
var GitCommit = "<unknown>"

// GitBranch comes from TRAVIS_BRANCH env variable - if it's github release, this variable will contain release tag name
var GitBranch = "<unknown>"

// BuildNumber comes from TRAVIS_JOB_NUMBER env variable
var BuildNumber = "dev-build"

// AsString returns all defined build constants as single string
func AsString() string {
	return fmt.Sprintf("Branch: %s. Build id: %s. Commit: %s", GitBranch, BuildNumber, GitCommit)
}

func GetInfo() *Info {
	return &Info{
		Commit:      GitCommit,
		Branch:      GitBranch,
		BuildNumber: BuildNumber,
	}
}
