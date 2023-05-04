/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package pkce

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPKCEInfo(t *testing.T) {
	_, err := New(42)
	assert.Error(t, err)

	_, err = New(129)
	assert.Error(t, err)

	for i := 43; i < 129; i++ {
		info, err := New(uint(i))
		assert.NoError(t, err)

		assert.NotEmpty(t, info.CodeVerifier)
		assert.NotEmpty(t, info.CodeChallenge)

		assert.Equal(t, i, len(info.CodeVerifier))

		assert.Equal(t, info.CodeChallenge, ChallengeSHA256(info.CodeVerifier))

		decoded, err := base64.RawURLEncoding.DecodeString(info.Base64URLCodeVerifier())
		assert.NoError(t, err)
		assert.Equal(t, info.CodeVerifier, string(decoded))
	}

}
