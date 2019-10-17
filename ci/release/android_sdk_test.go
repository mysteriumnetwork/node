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
	// given
	repositoryURL, _ := url.Parse("https://api.bintray.com/content/mysteriumnetwork/maven")
	r := releaseOpts{groupId: "network.mysterium", artifactId: "mobile-node", version: "0.10.0"}
	b := bintrayOpts{repositoryURL: repositoryURL}
	// when
	result := uploadURL(r, b, "build/package/Mysterium.aar")
	// then
	assert.Equal(t, "https://api.bintray.com/content/mysteriumnetwork/maven/network/mysterium/mobile-node/0.10.0/mobile-node-0.10.0.aar", result.String())
}

func Test_publishURL(t *testing.T) {
	// given
	repositoryURL, _ := url.Parse("https://api.bintray.com/content/mysteriumnetwork/maven")
	r := releaseOpts{groupId: "network.mysterium", artifactId: "mobile-node", version: "0.10.0"}
	b := bintrayOpts{repositoryURL: repositoryURL}
	// when
	result := publishURL(r, b)
	// then
	assert.Equal(t, "https://api.bintray.com/content/mysteriumnetwork/maven/network.mysterium:mobile-node/0.10.0/publish", result.String())
}
