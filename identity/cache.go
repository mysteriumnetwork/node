/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

type cacheData struct {
	Identity Identity `json:"identity"`
}

// IdentityCache saves identity to file
type IdentityCache struct {
	File string
}

// NewIdentityCache creates and returns identityCache
func NewIdentityCache(dir string, jsonFile string) IdentityCacheInterface {
	return &IdentityCache{
		File: filepath.Join(dir, jsonFile),
	}
}

// GetIdentity retrieves identity from cache
func (ic *IdentityCache) GetIdentity() (identity Identity, err error) {
	if !ic.cacheExists() {
		err = errors.New("cache file does not exist")
		return
	}

	cache, err := ic.readCache()
	if err != nil {
		return
	}

	return cache.Identity, nil
}

// StoreIdentity stores identity to cache
func (ic *IdentityCache) StoreIdentity(identity Identity) error {
	cache := cacheData{
		Identity: identity,
	}

	return ic.writeCache(cache)
}

func (ic *IdentityCache) cacheExists() bool {
	if _, err := os.Stat(ic.File); os.IsNotExist(err) {
		return false
	}

	return true
}

func (ic *IdentityCache) readCache() (cache *cacheData, err error) {
	data, err := ioutil.ReadFile(ic.File)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &cache)
	if err != nil {
		return
	}

	return
}

func (ic *IdentityCache) writeCache(cache cacheData) (err error) {
	cacheString, err := json.Marshal(cache)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(ic.File, cacheString, 0644)
	return
}
