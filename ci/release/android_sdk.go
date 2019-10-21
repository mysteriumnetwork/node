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
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/rs/zerolog/log"
)

const (
	// BintrayUsername user from https://bintray.com/mysteriumnetwork
	BintrayUsername = env.BuildVar("BINTRAY_USERNAME")
	// BintrayToken access token of the BintrayUsername
	BintrayToken = env.BuildVar("BINTRAY_TOKEN")
)

// ReleaseAndroidSDK releases Android SDK to Bintray
func ReleaseAndroidSDK() error {
	logconfig.Bootstrap()

	err := env.EnsureEnvVars(
		env.TagBuild,
		env.BuildVersion,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.TagBuild) {
		log.Info().Msg("Not a tag build, skipping ReleaseAndroidSDK action...")
		return nil
	}

	err = storage.DownloadArtifacts()
	if err != nil {
		return err
	}

	err = env.EnsureEnvVars(BintrayUsername, BintrayToken)
	if err != nil {
		return err
	}
	repositoryURL, err := url.Parse("https://api.bintray.com/content/mysteriumnetwork/maven")
	if err != nil {
		return err
	}
	uploader := newBintrayReleaser(
		&releaseOpts{
			groupId:    "network.mysterium",
			artifactId: "mobile-node",
			version:    env.Str(env.BuildVersion),
		},
		&bintrayOpts{
			repositoryURL: repositoryURL,
			username:      env.Str(BintrayUsername),
			password:      env.Str(BintrayToken),
		},
	)
	if err := uploader.upload("build/package/Mysterium.aar"); err != nil {
		return err
	}
	if err := uploader.upload("build/package/mvn.pom"); err != nil {
		return err
	}
	return uploader.publish()
}

type releaseOpts struct {
	groupId, artifactId, version string
}

type bintrayOpts struct {
	repositoryURL      *url.URL
	username, password string
}

// bintrayReleaser uploads and publishes new versions in Bintray via its REST API: https://bintray.com/docs/api/
type bintrayReleaser struct {
	client      *http.Client
	releaseOpts *releaseOpts
	bintrayOpts *bintrayOpts
}

func newBintrayReleaser(releaseOpts *releaseOpts, bintrayOpts *bintrayOpts) *bintrayReleaser {
	return &bintrayReleaser{
		client:      &http.Client{Timeout: 30 * time.Second},
		releaseOpts: releaseOpts,
		bintrayOpts: bintrayOpts,
	}
}

func (up *bintrayReleaser) upload(filename string) error {
	log.Info().Msg("Uploading: " + filename)

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	requestURL := uploadURL(*up.releaseOpts, *up.bintrayOpts, filename)
	return up.bintrayAPIRequest(http.MethodPut, *requestURL, file)
}

func (up *bintrayReleaser) publish() error {
	log.Info().Msg("Publishing version: " + up.releaseOpts.version)
	requestURL := publishURL(*up.releaseOpts, *up.bintrayOpts)
	return up.bintrayAPIRequest(http.MethodPost, *requestURL, nil)
}

func (up *bintrayReleaser) bintrayAPIRequest(method string, url url.URL, body io.Reader) error {
	log.Info().Msgf("Bintray request [%s] %s", method, url.String())
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(up.bintrayOpts.username, up.bintrayOpts.password)
	req.Header.Set("X-Bintray-Package", up.releaseOpts.groupId+":"+up.releaseOpts.artifactId)
	req.Header.Set("X-Bintray-Version", env.Str(env.BuildVersion))
	res, err := up.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resBytes, err := httputil.DumpResponse(res, true)
	if err != nil {
		return err
	}

	log.Info().Msgf("Bintray response: %v", string(resBytes))
	return nil
}

// uploadURL constructs bintray URL for artifact upload
func uploadURL(r releaseOpts, b bintrayOpts, filename string) *url.URL {
	segments := []string{b.repositoryURL.Path}
	segments = append(segments, strings.Split(r.groupId, ".")...)

	remoteFilename := r.artifactId + "-" + r.version + path.Ext(filename)
	segments = append(segments, r.artifactId, r.version, remoteFilename)

	resultURL, _ := url.Parse(b.repositoryURL.String())
	resultURL.Path = path.Join(segments...)
	return resultURL
}

// publishURL constructs bintray URL for publishing a version with uploaded artifacts.
// It is different from uploadURL in a way that it uses artifactId (e.g. network.mysterium:mobile-node) instead of a
// usual maven layout (e.g. network/mysterium/mobile-node).
func publishURL(r releaseOpts, b bintrayOpts) *url.URL {
	packageId := r.groupId + ":" + r.artifactId
	resultPath := path.Join(b.repositoryURL.Path, packageId, r.version, "publish")

	resultURL, _ := url.Parse(b.repositoryURL.String())
	resultURL.Path = resultPath
	return resultURL
}
