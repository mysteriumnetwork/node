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
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/google/go-github/github"
	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
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
	releaser, err := newGithubReleaser(owner, repo)
	if err != nil {
		return err
	}

	if err := storage.DownloadArtifacts(); err != nil {
		return err
	}
	releaseId, err := releaser.createRelease(ver)
	if err != nil {
		return err
	}

	artifactFilenames, err := ioutil.ReadDir("build/package")
	if err != nil {
		return err
	}
	for _, f := range artifactFilenames {
		p := path.Join("build/package", f.Name())
		err := releaser.uploadAsset(releaseId, p)
		if err != nil {
			return errors.Wrap(err, "could not upload artifact "+p)
		}
	}

	log.Info("artifacts uploaded successfully")
	return nil
}

type githubReleaser struct {
	client            *github.Client
	owner, repository string
}

func newGithubReleaser(owner, repo string) (*githubReleaser, error) {
	token, err := env.RequiredEnvStr(env.GithubApiToken)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(context.Background(), ts)
	return &githubReleaser{
		client:     github.NewClient(oauthClient),
		owner:      owner,
		repository: repo,
	}, nil
}

func (r *githubReleaser) createRelease(name string) (int64, error) {
	release, _, err := r.client.Repositories.CreateRelease(context.Background(), r.owner, r.repository, &github.RepositoryRelease{Name: github.String(name), TagName: github.String(name)})
	if err != nil {
		return 0, err
	}
	log.Info("created release ", *release.ID)
	return *release.ID, nil
}

func (r *githubReleaser) uploadAssets(releaseId int64, paths []string) error {
	for _, p := range paths {
		err := r.uploadAsset(releaseId, p)
		if err != nil {
			return err
		}
	}
	log.Info("artifacts uploaded successfully")
	return nil
}

func (r *githubReleaser) uploadAsset(releaseId int64, path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	asset, _, err := r.client.Repositories.UploadReleaseAsset(context.Background(), r.owner, r.repository, releaseId, &github.UploadOptions{
		Name: filepath.Base(file.Name()),
	}, file)
	if err != nil {
		return err
	}
	log.Info("uploaded asset ", *asset.Name)
	return nil
}
