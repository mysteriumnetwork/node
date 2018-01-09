package identity

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"errors"
)

type cacheData struct {
	Identity Identity `json:"identity"`
}

type IdentityCache struct {
	File string
}

func NewIdentityCache(dir string, jsonFile string) IdentityCacheInterface {
	return &IdentityCache{
		File: filepath.Join(dir, jsonFile),
	}
}

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
