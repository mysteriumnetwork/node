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

package github

import (
	"context"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	gogithub "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// Releaser releases to Github
type Releaser struct {
	client            *gogithub.Client
	owner, repository string
}

// Release represents a Github release
type Release struct {
	Id      int64
	TagName string
	*Releaser
}

// NewReleaser creates a new Releaser instance
func NewReleaser(owner, repo, token string) (*Releaser, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(context.Background(), ts)
	return &Releaser{
		client:     gogithub.NewClient(oauthClient),
		owner:      owner,
		repository: repo,
	}, nil
}

// Create creates a new Github release
func (r *Releaser) Create(name string) (*Release, error) {
	release, _, err := r.client.Repositories.CreateRelease(context.Background(), r.owner, r.repository, &gogithub.RepositoryRelease{Name: gogithub.String(name), TagName: gogithub.String(name)})
	if err != nil {
		return nil, err
	}
	log.Infof("created release ID: %v, tag: %v", *release.ID, *release.TagName)
	return &Release{Id: *release.ID, TagName: *release.TagName, Releaser: r}, nil
}

// Latest finds the latest Github release
func (r *Releaser) Latest() (*Release, error) {
	release, _, err := r.client.Repositories.GetLatestRelease(context.Background(), r.owner, r.repository)
	if err != nil {
		return nil, err
	}
	return &Release{Id: *release.ID, TagName: *release.TagName, Releaser: r}, nil
}

// UploadAsset uploads asset to the release
func (r *Release) UploadAsset(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	asset, _, err := r.client.Repositories.UploadReleaseAsset(context.Background(), r.owner, r.repository, r.Id, &gogithub.UploadOptions{
		Name: filepath.Base(file.Name()),
	}, file)
	if err != nil {
		return err
	}
	log.Info("uploaded asset ", *asset.Name)
	return nil
}
