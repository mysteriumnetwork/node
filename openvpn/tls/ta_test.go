/*
 * Copyright (C) 2018 The Mysterium Network Authors
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

package tls

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedKey = `-----BEGIN OpenVPN Static key V1-----
616263
-----END OpenVPN Static key V1-----
`

func TestTLSPresharedKeyProducesValidPEMFormat(t *testing.T) {
	key := TLSPresharedKey("abc")
	assert.Equal(
		t,
		expectedKey,
		key.ToPEMFormat(),
	)
}

func TestGeneratedKeyIsExpectedSize(t *testing.T) {
	key, err := createTLSCryptKey()
	assert.NoError(t, err)
	assert.Len(t, key, 256)
}
