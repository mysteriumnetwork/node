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
	// MavenToken token of the MavenUsername
	MavenToken = env.BuildVar("MAVEN_TOKEN")
)

// ReleaseAndroidSDKSnapshot releases Android SDK snapshot from master branch to maven repo
func ReleaseAndroidSDKSnapshot() error {
	logconfig.Bootstrap()

	err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.BuildVersion,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.SnapshotBuild) {
		log.Info().Msg("Not a snapshot build, skipping ReleaseAndroidSDKSnapshot action...")
		return nil
	}
	err = env.EnsureEnvVars(MavenToken)
	if err != nil {
		return err
	}
	repositoryURL, _ := url.Parse("https://maven.mysterium.network/snapshots")
	return releaseAndroidSDK(
		releaseOpts{
			groupId:    "network.mysterium",
			artifactId: "mobile-node",
			version:    env.Str(env.BuildVersion),
		},
		repositoryOpts{
			repositoryURL: repositoryURL,
			username:      "snapshots",
			password:      env.Str(MavenToken),
		},
	)
}

// ReleaseAndroidSDK releases tag Android SDK to maven repo
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

	err = env.EnsureEnvVars(MavenToken)
	if err != nil {
		return err
	}
	repositoryURL, _ := url.Parse("https://maven.mysterium.network/releases")
	return releaseAndroidSDK(
		releaseOpts{
			groupId:    "network.mysterium",
			artifactId: "mobile-node",
			version:    env.Str(env.BuildVersion),
		},
		repositoryOpts{
			repositoryURL: repositoryURL,
			username:      "releases",
			password:      env.Str(MavenToken),
		},
	)
}

func releaseAndroidSDK(rel releaseOpts, rep repositoryOpts) error {
	err := storage.DownloadArtifacts()
	if err != nil {
		return err
	}

	publisher := newMavenPublisher(&rel, &rep)
	if err := publisher.upload("build/package/Mysterium.aar"); err != nil {
		return err
	}
	if err := publisher.upload("build/package/mvn.pom"); err != nil {
		return err
	}
	return nil
}

type releaseOpts struct {
	groupId, artifactId, version string
}

type repositoryOpts struct {
	repositoryURL      *url.URL
	username, password string
}

// mavenPublisher uploads artifacts to maven repo
type mavenPublisher struct {
	client         *http.Client
	releaseOpts    *releaseOpts
	repositoryOpts *repositoryOpts
}

func newMavenPublisher(releaseOpts *releaseOpts, repositoryOpts *repositoryOpts) *mavenPublisher {
	return &mavenPublisher{
		client:         &http.Client{Timeout: 30 * time.Second},
		releaseOpts:    releaseOpts,
		repositoryOpts: repositoryOpts,
	}
}

func (up *mavenPublisher) upload(filename string) error {
	log.Info().Msg("Uploading: " + filename)

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	requestURL := uploadURL(*up.releaseOpts, *up.repositoryOpts, filename)
	return up.apiRequest(http.MethodPut, *requestURL, file)
}

func (up *mavenPublisher) apiRequest(method string, url url.URL, body io.Reader) error {
	log.Info().Msgf("API request [%s] %s", method, url.String())
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(up.repositoryOpts.username, up.repositoryOpts.password)
	res, err := up.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resBytes, err := httputil.DumpResponse(res, true)
	if err != nil {
		return err
	}

	log.Info().Msgf("API response: %v", string(resBytes))
	return nil
}

// uploadURL constructs URL for artifact upload
func uploadURL(r releaseOpts, b repositoryOpts, filename string) *url.URL {
	segments := []string{b.repositoryURL.Path}
	segments = append(segments, strings.Split(r.groupId, ".")...)

	remoteFilename := r.artifactId + "-" + r.version + path.Ext(filename)
	segments = append(segments, r.artifactId, r.version, remoteFilename)

	resultURL, _ := url.Parse(b.repositoryURL.String())
	resultURL.Path = path.Join(segments...)
	return resultURL
}
