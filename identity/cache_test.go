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

package identity

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var file = "/tmp/cache.json"

func TestIdentityCache_StoreIdentity(t *testing.T) {
	identity := FromAddress("0x000000000000000000000000000000000000000A")
	cache := IdentityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)
}

func TestIdentityCache_GetIdentity(t *testing.T) {
	identity := FromAddress("0x000000000000000000000000000000000000000A")
	cache := IdentityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)
	id, err := cache.GetIdentity()

	assert.Equal(t, id, identity)
	assert.Nil(t, err)
}

func TestIdentityCache_GetIdentityWithNoCache(t *testing.T) {
	cache := IdentityCache{
		File: "does-not-exist",
	}

	_, err := cache.GetIdentity()

	assert.Equal(t, errors.New("cache file does not exist"), err)
}

func TestIdentityCache_cacheExists(t *testing.T) {
	identity := FromAddress("0x000000000000000000000000000000000000000A")
	cache := IdentityCache{
		File: file,
	}

	err := cache.StoreIdentity(identity)
	assert.Nil(t, err)

	assert.True(t, cache.cacheExists())

	_, err = os.Stat(file)
	assert.True(t, err == nil && !os.IsNotExist(err))
}
