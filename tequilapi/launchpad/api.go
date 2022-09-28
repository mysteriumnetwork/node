/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package launchpad

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const versionNA = "N/A"

// ArchiveEntry launchpad archive entry
type ArchiveEntry struct {
	BinaryPackageVersion string `json:"binary_package_version"`
}

// ArchiveResponse launchpad archive response
type ArchiveResponse struct {
	Start     int            `json:"start"`
	TotalSize int            `json:"total_size"`
	Entries   []ArchiveEntry `json:"entries"`
}

type launchpadHTTP interface {
	Do(req *http.Request) (*http.Response, error)
}

// API - struct
type API struct {
	http  launchpadHTTP
	cache *cache
	lock  sync.Mutex
}

// New launchpad API instance
func New() *API {
	return &API{
		http:  &http.Client{Timeout: 20 * time.Second},
		cache: &cache{},
	}
}

// LatestPublishedReleaseVersion attempts to retrieve latest released binary semantic version
func (a *API) LatestPublishedReleaseVersion() (string, error) {
	response, err := a.latestPPABinaryReleases()
	if err != nil {
		return versionNA, err
	}

	if len(response.Entries) == 0 {
		return versionNA, err
	}

	latestPackageVersion := response.Entries[0].BinaryPackageVersion
	if len(latestPackageVersion) == 0 {
		return versionNA, err
	}

	split := strings.Split(latestPackageVersion, "+")
	if len(split) != 3 {
		return versionNA, err
	}

	return split[0], nil
}
func (a *API) latestPPABinaryReleases() (ArchiveResponse, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if !a.cache.expired() {
		return a.cache.get(), nil
	}

	req, err := http.NewRequest("GET", "https://api.launchpad.net/1.0/~mysteriumnetwork/+archive/ubuntu/node", nil)
	if err != nil {
		return ArchiveResponse{}, err
	}

	q := req.URL.Query()
	q.Add("ws.op", "getPublishedBinaries")
	q.Add("status", "Published")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Content-Type", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return ArchiveResponse{}, err
	}

	cr := ArchiveResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cr)
	a.cache.put(cr)
	return cr, nil
}

type cache struct {
	lock      sync.Mutex
	expiresAt time.Time
	response  ArchiveResponse
}

func (c *cache) get() ArchiveResponse {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.response
}

func (c *cache) put(response ArchiveResponse) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.expiresAt = time.Now().Add(20 * time.Minute)
	c.response = response
}

func (c *cache) expired() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.expiresAt.Before(time.Now())
}
