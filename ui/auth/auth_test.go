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

package auth

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseBasicAuthHeader(t *testing.T) {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:secret"))
	user, pass, err := parseBasicAuthHeader(header)
	assert.NoError(t, err)
	assert.Equal(t, "user", user)
	assert.Equal(t, "secret", pass)
}

func Test_parseBasicAuthHeaderIncorrect(t *testing.T) {
	header := "Basic " + base64.StdEncoding.EncodeToString([]byte("secretuser"))
	_, _, err := parseBasicAuthHeader(header)
	assert.EqualError(t, err, "incorrect basic auth header")

	header = "Basic " + base64.StdEncoding.EncodeToString([]byte("s:e:c:r:e:t:u:s:e:r"))
	_, _, err = parseBasicAuthHeader(header)
	assert.EqualError(t, err, "incorrect basic auth header")
}
