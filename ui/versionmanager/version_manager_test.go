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
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	godvpnweb "github.com/mysteriumnetwork/go-dvpn-web/v2"

	"github.com/stretchr/testify/assert"
)

var ms = &mockServer{}

type mockServer struct {
	capturedPath string
}

func (m *mockServer) SwitchUI(path string) {
	m.capturedPath = path
}

func TestSwitchUI(t *testing.T) {
	// given
	tmpDIr, err := os.MkdirTemp("", "nodeuiversiontest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDIr)

	err = os.Mkdir(tmpDIr+"/1.1.1", 0644)
	assert.NoError(t, err)

	config, err := NewVersionConfig(tmpDIr)
	assert.NoError(t, err)

	var nvm = &VersionManager{
		uiServer:      ms,
		versionConfig: config,
	}

	// when
	usedVersion, err := nvm.UsedVersion()

	// then
	assert.NoError(t, err)
	assert.Equal(t, BundledVersionName, usedVersion.Name)

	// when
	err = nvm.SwitchTo(BundledVersionName)

	// then
	assert.NoError(t, err)
	assert.Equal(t, BundledVersionName, ms.capturedPath)

	// when
	err = nvm.SwitchTo("non existing")

	// then
	assert.Error(t, err)
	assert.Equal(t, BundledVersionName, ms.capturedPath)

	// when
	err = nvm.SwitchTo("1.1.1")
	assert.NoError(t, err)

	usedVersion, err = nvm.UsedVersion()
	assert.NoError(t, err)

	// then
	assert.Equal(t, tmpDIr+"/1.1.1/build", ms.capturedPath)
	assert.Equal(t, "1.1.1", usedVersion.Name)
}

func TestListLocal(t *testing.T) {
	// given
	tmpDIr, err := os.MkdirTemp("", "nodeuiversiontest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDIr)

	config, err := NewVersionConfig(tmpDIr)
	assert.NoError(t, err)

	var nvm = &VersionManager{
		uiServer:      ms,
		versionConfig: config,
	}

	// when
	versions, err := nvm.ListLocalVersions()
	assert.NoError(t, err)

	// then
	assert.Len(t, versions, 0)

	// when
	err = os.Mkdir(tmpDIr+"/1.1.1", 0644)
	assert.NoError(t, err)
	err = os.Mkdir(tmpDIr+"/1.2.2", 0644)
	assert.NoError(t, err)

	versions, err = nvm.ListLocalVersions()

	// then
	for _, version := range []string{"1.1.1", "1.2.2"} {
		assert.Contains(t, versions, LocalVersion{Name: version})
	}
}

func TestBundledVersion(t *testing.T) {
	// given
	tmpDIr, err := os.MkdirTemp("", "nodeuiversiontest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDIr)

	bundledNodeUIVersion, err := godvpnweb.Version()
	assert.NoError(t, err)

	config, err := NewVersionConfig(tmpDIr)
	assert.NoError(t, err)

	var nvm = &VersionManager{
		uiServer:      ms,
		versionConfig: config,
	}

	// when
	version, err := nvm.BundledVersion()

	// then
	assert.NoError(t, err)
	assert.Equal(t, bundledNodeUIVersion, version.Name)
}

func TestListLocalDirectoryRemoved(t *testing.T) {
	// given
	tmpDIr, err := os.MkdirTemp("", "nodeuiversiontest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDIr)

	config, err := NewVersionConfig(tmpDIr)
	assert.NoError(t, err)

	var nvm = &VersionManager{
		uiServer:      ms,
		versionConfig: config,
	}

	// when
	err = os.RemoveAll(tmpDIr)
	assert.NoError(t, err)

	versions, err := nvm.ListLocalVersions()

	// then
	assert.Len(t, versions, 0)
	assert.NoError(t, err)
}

func TestListRemoteVersions(t *testing.T) {
	// given
	rMap := map[string]*http.Response{
		"/repos/mysteriumnetwork/dvpn-web/releases": {StatusCode: 200, Status: "200 OK", Body: &readCloser{Reader: strings.NewReader(releasesJSON)}},
	}
	httpMock := newHttpClientMock(rMap)

	var nvm = &VersionManager{
		uiServer: ms,
		github:   newGithub(httpMock),
	}

	// expect
	assert.Empty(t, nvm.releasesCache)

	// when
	versionsBeforeFlush, err := nvm.ListRemoteVersions(RemoteVersionRequest{})
	assert.NoError(t, err)

	// then
	assert.NotEmpty(t, nvm.releasesCache)
	assert.Equal(t, len(versionsBeforeFlush), len(nvm.releasesCache))

	// when
	httpMock.stub(map[string]*http.Response{
		"/repos/mysteriumnetwork/dvpn-web/releases": {StatusCode: 200, Status: "200 OK", Body: &readCloser{Reader: strings.NewReader(singleReleaseJSON)}},
	})
	versionsBeforeFlush, err = nvm.ListRemoteVersions(RemoteVersionRequest{})
	assert.NoError(t, err)

	// then
	assert.NotEmpty(t, nvm.releasesCache)
	assert.Equal(t, len(versionsBeforeFlush), len(nvm.releasesCache))

	// when
	versionsFlushed, err := nvm.ListRemoteVersions(RemoteVersionRequest{FlushCache: true})
	assert.NoError(t, err)
	assert.NotEqual(t, len(versionsBeforeFlush), len(nvm.releasesCache))
	assert.Len(t, nvm.releasesCache, len(versionsFlushed))

	// and
	first := versionsFlushed[0]
	assert.Equal(t, "Release Notes", first.ReleaseNotes)
	assert.Equal(t, "1.0.1", first.Name)

	expectedTime, err := time.Parse("2006-01-02T15:04:05Z", "2021-11-22T13:17:45Z")
	assert.NoError(t, err)

	assert.Equal(t, expectedTime, first.PublishedAt)
	assert.False(t, first.IsPreRelease)
}

var singleReleaseJSON = `[
{
		"url": "https://api.github.com/repos/mysteriumnetwork/dvpn-web/releases/53839105",
		"assets_url": "https://api.github.com/repos/mysteriumnetwork/dvpn-web/releases/53839105/assets",
		"upload_url": "https://uploads.github.com/repos/mysteriumnetwork/dvpn-web/releases/53839105/assets{?name,label}",
		"html_url": "https://github.com/mysteriumnetwork/dvpn-web/releases/tag/1.0.1",
		"id": 53839105,
		"author": {
			"login": "mdomasevicius",
			"id": 20511486,
			"node_id": "MDQ6VXNlcjIwNTExNDg2",
			"avatar_url": "https://avatars.githubusercontent.com/u/20511486?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/mdomasevicius",
			"html_url": "https://github.com/mdomasevicius",
			"followers_url": "https://api.github.com/users/mdomasevicius/followers",
			"following_url": "https://api.github.com/users/mdomasevicius/following{/other_user}",
			"gists_url": "https://api.github.com/users/mdomasevicius/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/mdomasevicius/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/mdomasevicius/subscriptions",
			"organizations_url": "https://api.github.com/users/mdomasevicius/orgs",
			"repos_url": "https://api.github.com/users/mdomasevicius/repos",
			"events_url": "https://api.github.com/users/mdomasevicius/events{/privacy}",
			"received_events_url": "https://api.github.com/users/mdomasevicius/received_events",
			"type": "User",
			"site_admin": false
		},
		"node_id": "RE_kwDOCwMSYc4DNYUB",
		"tag_name": "1.0.1",
		"target_commitish": "mainnet",
		"name": "1.0.1",
		"draft": false,
		"prerelease": false,
		"created_at": "2021-11-22T13:16:34Z",
		"published_at": "2021-11-22T13:17:45Z",
		"assets": [
			{
				"url": "https://api.github.com/repos/mysteriumnetwork/dvpn-web/releases/assets/50043500",
				"id": 50043500,
				"node_id": "RA_kwDOCwMSYc4C-5ps",
				"name": "dist.tar.gz",
				"label": "",
				"uploader": {
					"login": "MysteriumTeam",
					"id": 42934344,
					"node_id": "MDQ6VXNlcjQyOTM0MzQ0",
					"avatar_url": "https://avatars.githubusercontent.com/u/42934344?v=4",
					"gravatar_id": "",
					"url": "https://api.github.com/users/MysteriumTeam",
					"html_url": "https://github.com/MysteriumTeam",
					"followers_url": "https://api.github.com/users/MysteriumTeam/followers",
					"following_url": "https://api.github.com/users/MysteriumTeam/following{/other_user}",
					"gists_url": "https://api.github.com/users/MysteriumTeam/gists{/gist_id}",
					"starred_url": "https://api.github.com/users/MysteriumTeam/starred{/owner}{/repo}",
					"subscriptions_url": "https://api.github.com/users/MysteriumTeam/subscriptions",
					"organizations_url": "https://api.github.com/users/MysteriumTeam/orgs",
					"repos_url": "https://api.github.com/users/MysteriumTeam/repos",
					"events_url": "https://api.github.com/users/MysteriumTeam/events{/privacy}",
					"received_events_url": "https://api.github.com/users/MysteriumTeam/received_events",
					"type": "User",
					"site_admin": false
				},
				"content_type": "application/octet-stream",
				"state": "uploaded",
				"size": 1515748,
				"download_count": 3,
				"created_at": "2021-11-22T13:26:23Z",
				"updated_at": "2021-11-22T13:26:23Z",
				"browser_download_url": "https://github.com/mysteriumnetwork/dvpn-web/releases/download/1.0.1/dist.tar.gz"
			}
		],
		"tarball_url": "https://api.github.com/repos/mysteriumnetwork/dvpn-web/tarball/1.0.1",
		"zipball_url": "https://api.github.com/repos/mysteriumnetwork/dvpn-web/zipball/1.0.1",
		"body": "Release Notes", 
		"mentions_count": 3
	}
]
`
