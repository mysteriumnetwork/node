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

package env

// BuildVar env variable required for CI build.
// Some of them will be calculated when generating env file, others should be passed though env.
type BuildVar string

const (
	// TagBuild indicates release build
	TagBuild = BuildVar("RELEASE_BUILD")

	// SnapshotBuild indicates snapshot release build (master branch)
	SnapshotBuild = BuildVar("SNAPSHOT_BUILD")

	// PrBuild indicates pull-request build
	PrBuild = BuildVar("PR_BUILD")

	// BuildVersion stores build version
	BuildVersion = BuildVar("BUILD_VERSION")

	// PpaVersion stores build version for PPA
	PpaVersion = BuildVar("PPA_VERSION")

	// BuildNumber stores CI build number
	BuildNumber = BuildVar("BUILD_NUMBER")

	// GithubOwner stores github repository's owner
	GithubOwner = BuildVar("GITHUB_OWNER")

	// GithubRepository stores github repository name
	GithubRepository = BuildVar("GITHUB_REPO")

	// GithubSnapshotRepository stores github repository name for snapshot builds
	GithubSnapshotRepository = BuildVar("GITHUB_SNAPSHOT_REPO")

	// GithubApiToken is used for accessing github API
	GithubApiToken = BuildVar("GITHUB_API_TOKEN")
)
