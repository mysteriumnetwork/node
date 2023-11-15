/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package versionmanager

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	godvpnweb "github.com/mysteriumnetwork/go-dvpn-web/v2"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NodeUIServer interface with UI server
type NodeUIServer interface {
	SwitchUI(path string)
}

// VersionManager Node UI version manager
type VersionManager struct {
	uiServer      NodeUIServer
	httpClient    *requests.HTTPClient
	github        *github
	versionConfig NodeUIVersionConfig

	remoteCacheExpiresAt time.Time
	releasesCache        []GitHubRelease

	downloader *Downloader
}

// NewVersionManager VersionManager constructor
func NewVersionManager(
	uiServer NodeUIServer,
	http *requests.HTTPClient,
	versionConfig NodeUIVersionConfig,
) *VersionManager {
	return &VersionManager{
		uiServer:      uiServer,
		httpClient:    http,
		versionConfig: versionConfig,
		github:        newGithub(http),
		downloader:    NewDownloader(),
	}
}

// TODO check integrity of downloaded release so not to serve a broken nodeUI

// ListLocalVersions list downloaded Node UI versions
func (vm *VersionManager) ListLocalVersions() ([]LocalVersion, error) {
	var versions = make([]LocalVersion, 0)

	files, err := os.ReadDir(vm.versionConfig.uiDir())
	if err != nil {
		if os.IsNotExist(err) {
			return versions, nil
		}
		return nil, fmt.Errorf("could not read "+nodeUIPath+": %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			versions = append(versions, LocalVersion{
				Name: f.Name(),
			})
		}
	}

	return versions, nil
}

// UsedVersion current version
func (vm *VersionManager) UsedVersion() (LocalVersion, error) {
	version, err := vm.versionConfig.Version()
	if err != nil {
		return LocalVersion{}, err
	}
	return LocalVersion{
		Name: version,
	}, nil
}

// BundledVersion bundled version
func (vm *VersionManager) BundledVersion() (LocalVersion, error) {
	name, err := godvpnweb.Version()
	if err != nil {
		return LocalVersion{}, err
	}
	return LocalVersion{
		Name: name,
	}, nil
}

// RemoteVersionRequest for paged requests
type RemoteVersionRequest struct {
	PerPage    int
	Page       int
	FlushCache bool
}

// ListRemoteVersions list versions from github releases of NodeUI
func (vm *VersionManager) ListRemoteVersions(r RemoteVersionRequest) ([]RemoteVersion, error) {
	if time.Now().Before(vm.remoteCacheExpiresAt) && vm.releasesCache != nil && !r.FlushCache {
		return remoteVersions(vm.releasesCache), nil
	}

	releases, err := vm.github.nodeUIReleases(r.PerPage, r.Page)
	if err != nil {
		return nil, err
	}

	vm.releasesCache = releases
	vm.remoteCacheExpiresAt = time.Now().Add(time.Hour)

	return remoteVersions(vm.releasesCache), nil
}

func remoteVersions(releases []GitHubRelease) []RemoteVersion {
	var versions = make([]RemoteVersion, 0)
	for _, release := range releases {
		versions = append(versions, RemoteVersion{
			Name:             release.Name,
			PublishedAt:      release.PublishedAt,
			CompatibilityURL: compatibilityAssetURL(release),
			IsPreRelease:     release.Prerelease,
			ReleaseNotes:     release.Body,
		})
	}
	return versions
}

func compatibilityAssetURL(r GitHubRelease) string {
	for _, a := range r.Assets {
		if a.Name == compatibilityAssetName {
			return a.BrowserDownloadUrl
		}
	}
	return ""
}

// TODO think about sending SSE to inform to nodeUI that a version has been downloaded

// Download and untar node UI dist
func (vm *VersionManager) Download(versionName string) error {
	assetURL, err := vm.github.nodeUIDownloadURL(versionName)
	if err != nil {
		return err
	}

	err = os.MkdirAll(vm.versionConfig.uiDistPath(versionName), 0700)
	if err != nil {
		return err
	}

	vm.downloader.DownloadNodeUI(DownloadOpts{
		URL:      assetURL,
		Tag:      versionName,
		DistFile: vm.versionConfig.uiDistFile(versionName),
		Callback: func(opts DownloadOpts) error {
			return vm.untarAndExplode(versionName)
		},
	})

	return nil
}

// SwitchTo switch to serving specific version
func (vm *VersionManager) SwitchTo(versionName string) error {
	log.Info().Msgf("Switching node UI to version: %s", versionName)

	if versionName == BundledVersionName {
		if err := vm.versionConfig.write(nodeUIVersion{VersionName: BundledVersionName}); err != nil {
			return err
		}
		vm.uiServer.SwitchUI(BundledVersionName)
		return nil
	}

	local, err := vm.ListLocalVersions()
	if err != nil {
		return err
	}
	for _, lv := range local {
		if lv.Name == versionName {
			if err := vm.versionConfig.write(nodeUIVersion{VersionName: versionName}); err != nil {
				return err
			}
			vm.uiServer.SwitchUI(vm.versionConfig.UIBuildPath(versionName))
			return nil
		}
	}

	return errors.New("no local version named: " + versionName)
}

func (vm *VersionManager) untarAndExplode(versionName string) error {
	file, err := os.Open(vm.versionConfig.uiDistFile(versionName))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	err = untar(vm.versionConfig.uiDistPath(versionName), file)
	if err != nil {
		return fmt.Errorf("failed to untar nodeUI dist: %w", err)
	}

	return nil
}

// DownloadStatus provides download status
func (vm *VersionManager) DownloadStatus() Status {
	return vm.downloader.Status()
}

// LocalVersion it's a local version with extra indicator if it is in use
type LocalVersion struct {
	Name string `json:"name"`
}

// RemoteVersion it's a version
type RemoteVersion struct {
	Name             string    `json:"name"`
	PublishedAt      time.Time `json:"released_at"`
	CompatibilityURL string    `json:"compatibility_url,omitempty"`
	IsPreRelease     bool      `json:"is_pre_release"`
	ReleaseNotes     string    `json:"release_notes,omitempty"`
}

func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0700); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}
