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
	"io/ioutil"
	"path"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/mysteriumnetwork/node/ci/github"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
)

// ReleaseSnapshot releases snapshot build
func ReleaseSnapshot() error {
	logconfig.Bootstrap()

	snapshot, err := env.RequiredEnvBool(env.SnapshotBuild)
	if err != nil {
		return err
	}
	if !snapshot {
		log.Info("not a snapshot build, skipping ReleaseSnapshot action...")
		return nil
	}

	owner, err := env.RequiredEnvStr(env.GithubOwner)
	if err != nil {
		return err
	}
	repo, err := env.RequiredEnvStr(env.GithubSnapshotRepository)
	if err != nil {
		return err
	}
	ver, err := env.RequiredEnvStr(env.BuildVersion)
	if err != nil {
		return err
	}
	githubToken, err := env.RequiredEnvStr(env.GithubApiToken)
	if err != nil {
		return err
	}
	releaser, err := github.NewReleaser(owner, repo, githubToken)
	if err != nil {
		return err
	}

	if err := storage.DownloadArtifacts(); err != nil {
		return err
	}
	release, err := releaser.Create(ver)
	if err != nil {
		return err
	}

	artifactFilenames, err := ioutil.ReadDir("build/package")
	if err != nil {
		return err
	}
	for _, f := range artifactFilenames {
		p := path.Join("build/package", f.Name())
		err := release.UploadAsset(p)
		if err != nil {
			return errors.Wrap(err, "could not upload artifact "+p)
		}
	}

	log.Info("artifacts uploaded successfully")
	return nil
}
