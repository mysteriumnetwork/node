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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_uploadURL(t *testing.T) {
	tests := []struct {
		name              string
		file              string
		releaseOpts       releaseOpts
		repositoryURL     string
		expectedUploadURL string
	}{
		{
			name: "artifact",
			file: "build/package/Mysterium.aar",
			releaseOpts: releaseOpts{
				groupId:    "network.mysterium",
				artifactId: "mobile-node",
				version:    "0.40.4-pre",
			},
			repositoryURL:     "https://maven.mysterium.network/releases",
			expectedUploadURL: "https://maven.mysterium.network/releases/network/mysterium/mobile-node/0.40.4-pre/mobile-node-0.40.4-pre.aar",
		},
		{
			name: "pom artifact",
			file: "build/package/mvn.pom",
			releaseOpts: releaseOpts{
				groupId:    "network.mysterium",
				artifactId: "mobile-node",
				version:    "0.40.4-pre",
			},
			repositoryURL:     "https://maven.mysterium.network/releases",
			expectedUploadURL: "https://maven.mysterium.network/releases/network/mysterium/mobile-node/0.40.4-pre/mobile-node-0.40.4-pre.pom",
		},
		{
			name: "snapshot artifact",
			file: "build/package/Mysterium.aar",
			releaseOpts: releaseOpts{
				groupId:    "network.mysterium",
				artifactId: "mobile-node",
				version:    "0.40.4-11111",
			},
			repositoryURL:     "https://maven.mysterium.network/snapshots",
			expectedUploadURL: "https://maven.mysterium.network/snapshots/network/mysterium/mobile-node/0.40.4-11111/mobile-node-0.40.4-11111.aar",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			repositoryURL, _ := url.Parse(test.repositoryURL)
			// when
			result := uploadURL(test.releaseOpts, repositoryOpts{repositoryURL: repositoryURL}, test.file)
			// then
			assert.Equal(t, test.expectedUploadURL, result.String())
		})
	}
}
