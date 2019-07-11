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

package release

import (
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/shell"
)

type releaseDebianOpts struct {
	repository  string
	version     string
	buildNumber string
}

func releaseDebianPPA(opts *releaseDebianOpts) error {
	err := shell.NewCmdf("bin/release_ppa %s %s %s %s", opts.repository, opts.version, opts.buildNumber, "xenial").Run()
	if err != nil {
		return err
	}
	err = shell.NewCmdf("bin/release_ppa %s %s %s %s", opts.repository, opts.version, opts.buildNumber, "bionic").Run()
	if err != nil {
		return err
	}
	return nil
}

func ppaVersion(buildVersion string) string {
	// PPA treats minus as previous version
	return strings.Replace(buildVersion, "-", "+", -1)
}

// ReleaseDebianPPASnapshot releases to node-dev PPA
func ReleaseDebianPPASnapshot() error {
	err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.BuildVersion,
		env.BuildNumber,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.SnapshotBuild) {
		log.Info("not a snapshot build, skipping ReleaseDebianPPASnapshot action...")
		return nil
	}
	return releaseDebianPPA(&releaseDebianOpts{
		repository:  "node-dev",
		version:     ppaVersion(env.Str(env.BuildVersion)),
		buildNumber: env.Str(env.BuildNumber),
	})
}

// ReleaseDebianPPAPreRelease releases to node-pre PPA (which is then manually promoted to node PPA)
func ReleaseDebianPPAPreRelease() error {
	err := env.EnsureEnvVars(
		env.TagBuild,
		env.BuildVersion,
		env.BuildNumber,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.TagBuild) {
		log.Info("not a tag build, skipping ReleaseDebianPPAPreRelease action...")
		return nil
	}

	return releaseDebianPPA(&releaseDebianOpts{
		repository:  "node-pre",
		version:     ppaVersion(env.Str(env.BuildVersion)),
		buildNumber: env.Str(env.BuildNumber),
	})
}
