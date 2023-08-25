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
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/mysteriumnetwork/node/ci/packages"
	"github.com/mysteriumnetwork/node/logconfig"
)

// ReleaseDockerSnapshot uploads docker snapshot images to myst snapshots repository in docker hub
func ReleaseDockerSnapshot() error {
	logconfig.Bootstrap()

	if err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.BuildVersion,
	); err != nil {
		return err
	}
	job.Precondition(func() bool {
		return env.Bool(env.SnapshotBuild)
	})

	return packages.BuildMystAlpineImage(
		[]string{"mysteriumnetwork/myst-snapshots:" + env.Str(env.BuildVersion) + "-alpine", "mysteriumnetwork/myst-snapshots:latest"},
		true,
	)
}

// ReleaseDockerTag uploads docker tag release images to docker hub
func ReleaseDockerTag() error {
	logconfig.Bootstrap()

	if err := env.EnsureEnvVars(
		env.TagBuild,
		env.RCBuild,
		env.BuildVersion,
	); err != nil {
		return err
	}
	job.Precondition(func() bool {
		return env.Bool(env.TagBuild)
	})

	if env.Bool(env.RCBuild) {
		err := packages.BuildMystAlpineImage(
			[]string{"mysteriumnetwork/myst:" + env.Str(env.BuildVersion) + "-alpine"},
			true,
		)
		if err != nil {
			return err
		}

		err = packages.BuildMystDocumentationImage(
			[]string{"mysteriumnetwork/documentation:" + env.Str(env.BuildVersion)},
			true,
		)
		if err != nil {
			return err
		}
	} else {
		err := packages.BuildMystAlpineImage(
			[]string{
				"mysteriumnetwork/myst:" + env.Str(env.BuildVersion) + "-alpine",
				"mysteriumnetwork/myst:latest-alpine",
				"mysteriumnetwork/myst:latest",
			},
			true,
		)
		if err != nil {
			return err
		}

		err = packages.BuildMystDocumentationImage(
			[]string{
				"mysteriumnetwork/documentation:" + env.Str(env.BuildVersion),
				"mysteriumnetwork/documentation:latest",
			},
			true,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
