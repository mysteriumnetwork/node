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
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLaunchpadAPI struct {
}

func (m *mockLaunchpadAPI) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(responseJSON)),
	}, nil
}

func TestLocalAPIServerPortIsAsExpected(t *testing.T) {
	// given
	mockAPI := mockLaunchpadAPI{}
	c := cache{}
	api := API{
		cache: &c,
		http:  &mockAPI,
	}

	// expect
	assert.True(t, c.expiresAt.IsZero())
	assert.Equal(t, 0, len(c.response.Entries))

	// when
	version, err := api.LatestPublishedReleaseVersion()
	assert.NoError(t, err)
	assert.Equal(t, "1.17.4", version)

	// expect
	assert.True(t, !c.expiresAt.IsZero())
	assert.Equal(t, 1, len(c.response.Entries))
	assert.Equal(t, "1.17.4+build648396996+jammy", c.response.Entries[0].BinaryPackageVersion)
}

type mockHTTPClient struct {
}

const responseJSON = `{
	"start": 0,
	"total_size": 1,
	"entries": [
		{
			"self_link": "https://api.launchpad.net/1.0/~mysteriumnetwork/+archive/ubuntu/node/+binarypub/181640873",
			"resource_type_link": "https://api.launchpad.net/1.0/#binary_package_publishing_history",
			"display_name": "myst 1.17.4+build648396996+jammy in jammy amd64",
			"component_name": "main",
			"section_name": "devel",
			"source_package_name": "myst",
			"source_package_version": "1.17.4+build648396996+jammy",
			"distro_arch_series_link": "https://api.launchpad.net/1.0/ubuntu/jammy/amd64",
			"phased_update_percentage": null,
			"date_published": "2022-09-26T07:33:30.656503+00:00",
			"scheduled_deletion_date": null,
			"status": "Published",
			"pocket": "Release",
			"creator_link": null,
			"date_created": "2022-09-26T07:16:19.827227+00:00",
			"date_superseded": null,
			"date_made_pending": null,
			"date_removed": null,
			"archive_link": "https://api.launchpad.net/1.0/~mysteriumnetwork/+archive/ubuntu/node",
			"copied_from_archive_link": "https://api.launchpad.net/1.0/~mysteriumnetwork/+archive/ubuntu/node-pre",
			"removed_by_link": null,
			"removal_comment": null,
			"binary_package_name": "myst",
			"binary_package_version": "1.17.4+build648396996+jammy",
			"build_link": "https://api.launchpad.net/1.0/~mysteriumnetwork/+archive/ubuntu/node-pre/+build/24496120",
			"architecture_specific": true,
			"priority_name": "OPTIONAL",
			"http_etag": "\"b96c547d46dd408be77abacfebd555f2e927927b-c4489c5b704cdb2055d97c780aeaf46f74b72c73\""
		}
	]
}`
