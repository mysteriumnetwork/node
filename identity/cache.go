package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
	"io/ioutil"
	"encoding/json"
	"os"
	"path/filepath"
)

type cacheData struct {
	Identity dto.Identity `json:"identity"`
}

type identityCache struct {
	File string
}

func NewIdentityCache(dir string, jsonFile string) *identityCache {
	return &identityCache{
		File: filepath.Join(dir, jsonFile),
	}
}

func (ic *identityCache) GetIdentity() *dto.Identity {
	if ic.cacheExists() {
		cache, err := ic.readCache()
		if err != nil {
			return nil
		}

		return &cache.Identity
	}

	return nil
}

func (ic *identityCache) StoreIdentity(identity *dto.Identity) (error) {
	cache := cacheData{
		Identity: *identity,
	}

	return ic.writeCache(cache)
}

func (ic *identityCache) cacheExists() bool {
	if _, err := os.Stat(ic.File); os.IsNotExist(err) {
		return false
	}

	return true
}

func (ic *identityCache) readCache() (cache *cacheData, err error) {
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

func (ic *identityCache) writeCache(cache cacheData) (err error) {
	cacheString, err := json.Marshal(cache)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(ic.File, cacheString, 0644)
	return
}
