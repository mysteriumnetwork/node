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
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mysteriumnetwork/node/requests"
)

const (
	nodeUIAssetName        = "dist.tar.gz"
	compatibilityAssetName = "compatibility.json"
	apiURI                 = "https://api.github.com"
	nodeUIPath             = "repos/mysteriumnetwork/dvpn-web"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
	DoRequestAndParseResponse(req *http.Request, resp interface{}) error
}

type github struct {
	http httpClient
}

func newGithub(httpClient httpClient) *github {
	return &github{http: httpClient}
}

func (g *github) nodeUIReleases(perPage int, page int) ([]GitHubRelease, error) {
	req, err := requests.NewGetRequest(apiURI, fmt.Sprintf("%s/releases", nodeUIPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NodeUI releases fetch request: %w", err)
	}

	q := req.URL.Query()
	if perPage != 0 {
		q.Add("per_page", fmt.Sprint(perPage))
	}

	if page != 0 {
		q.Add("page", fmt.Sprint(page))
	}
	req.URL.RawQuery = q.Encode()

	res, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NodeUI releases: %w", err)
	}

	if err := requests.ParseResponseError(res); err != nil {
		return nil, fmt.Errorf("response error: %w", err)
	}

	var releases []GitHubRelease
	err = requests.ParseResponseJSON(res, &releases)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return releases, nil
}

func (g *github) nodeUIReleaseByVersion(name string) (GitHubRelease, error) {
	req, err := requests.NewGetRequest(apiURI, fmt.Sprintf("%s/releases/tags/%s", nodeUIPath, name), nil)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("could not create request: %w", err)
	}

	var release GitHubRelease
	err = g.http.DoRequestAndParseResponse(req, &release)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("could not fetch version tagged %q: %w", name, err)
	}

	return release, nil
}

func (g *github) nodeUIDownloadURL(versionName string) (*url.URL, error) {
	r, err := g.nodeUIReleaseByVersion(versionName)
	if err != nil {
		return nil, err
	}

	req, err := requests.NewGetRequest(apiURI, fmt.Sprintf("%s/releases/%d/assets", nodeUIPath, r.Id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NodeUI assets ID %d, versionName %q: %w", r.Id, r.Name, err)
	}

	var assets []GithubAsset
	err = g.http.DoRequestAndParseResponse(req, &assets)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NodeUI releases: %w", err)
	}

	uiDist, ok := findNodeUIDist(assets)
	if !ok {
		return nil, fmt.Errorf("could not find nodeUI dist asset for release ID %d version %s", r.Id, versionName)
	}
	return url.Parse(uiDist.BrowserDownloadUrl)
}

func findNodeUIDist(assets []GithubAsset) (GithubAsset, bool) {
	for _, ass := range assets {
		if ass.Name == nodeUIAssetName {
			return ass, true
		}
	}
	return GithubAsset{}, false
}

// GitHubRelease github release resource
type GitHubRelease struct {
	Url       string `json:"url"`
	AssetsUrl string `json:"assets_url"`
	UploadUrl string `json:"upload_url"`
	HtmlUrl   string `json:"html_url"`
	Id        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		Id                int    `json:"id"`
		NodeId            string `json:"node_id"`
		AvatarUrl         string `json:"avatar_url"`
		GravatarId        string `json:"gravatar_id"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeId          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		Url      string `json:"url"`
		Id       int    `json:"id"`
		NodeId   string `json:"node_id"`
		Name     string `json:"name"`
		Label    string `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			Id                int    `json:"id"`
			NodeId            string `json:"node_id"`
			AvatarUrl         string `json:"avatar_url"`
			GravatarId        string `json:"gravatar_id"`
			Url               string `json:"url"`
			HtmlUrl           string `json:"html_url"`
			FollowersUrl      string `json:"followers_url"`
			FollowingUrl      string `json:"following_url"`
			GistsUrl          string `json:"gists_url"`
			StarredUrl        string `json:"starred_url"`
			SubscriptionsUrl  string `json:"subscriptions_url"`
			OrganizationsUrl  string `json:"organizations_url"`
			ReposUrl          string `json:"repos_url"`
			EventsUrl         string `json:"events_url"`
			ReceivedEventsUrl string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadUrl string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballUrl string `json:"tarball_url"`
	ZipballUrl string `json:"zipball_url"`
	Body       string `json:"body"`
}

// GithubAsset github asset resource
type GithubAsset struct {
	Url      string `json:"url"`
	Id       int    `json:"id"`
	NodeId   string `json:"node_id"`
	Name     string `json:"name"`
	Label    string `json:"label"`
	Uploader struct {
		Login             string `json:"login"`
		Id                int    `json:"id"`
		NodeId            string `json:"node_id"`
		AvatarUrl         string `json:"avatar_url"`
		GravatarId        string `json:"gravatar_id"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"uploader"`
	ContentType        string    `json:"content_type"`
	State              string    `json:"state"`
	Size               int       `json:"size"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadUrl string    `json:"browser_download_url"`
}
