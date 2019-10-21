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

	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/github"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type releaseGithubOpts struct {
	owner      string
	repository string
	version    string
	token      string
	createTag  bool
}

// release releases build/package files to github
func releaseGithub(opts *releaseGithubOpts) error {
	releaser, err := github.NewReleaser(opts.owner, opts.repository, opts.token)
	if err != nil {
		return err
	}
	if err := storage.DownloadArtifacts(); err != nil {
		return err
	}

	var release *github.Release
	if opts.createTag {
		release, err = releaser.Create(opts.version)
	} else {
		release, err = releaser.Find(opts.version)
	}
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

	log.Info().Msg("Artifacts uploaded successfully")
	return nil
}

// ReleaseGithubSnapshot releases snapshot to github
func ReleaseGithubSnapshot() error {
	logconfig.Bootstrap()

	err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.GithubOwner,
		env.GithubSnapshotRepository,
		env.BuildVersion,
		env.GithubAPIToken,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.SnapshotBuild) {
		log.Info().Msg("Not a snapshot build, skipping ReleaseGithubSnapshot action...")
		return nil
	}

	return releaseGithub(&releaseGithubOpts{
		owner:      env.Str(env.GithubOwner),
		repository: env.Str(env.GithubSnapshotRepository),
		version:    env.Str(env.BuildVersion),
		token:      env.Str(env.GithubAPIToken),
		createTag:  true,
	})
}

// ReleaseGithubTag releases tag to github
func ReleaseGithubTag() error {
	logconfig.Bootstrap()

	err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.GithubOwner,
		env.GithubRepository,
		env.BuildVersion,
		env.GithubAPIToken,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.TagBuild) {
		log.Info().Msg("Not a tag build, skipping ReleaseGithubTag action...")
		return nil
	}

	return releaseGithub(&releaseGithubOpts{
		owner:      env.Str(env.GithubOwner),
		repository: env.Str(env.GithubRepository),
		version:    env.Str(env.BuildVersion),
		token:      env.Str(env.GithubAPIToken),
		createTag:  false, // Tag is already created manually - which is release process trigger
	})
}
