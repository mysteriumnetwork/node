package identity

import "github.com/mysterium/node/service_discovery/dto"

type identityController struct {
	manager *identityManager
	cache   *identityCache
}

func NewIdentityController(dir string) *identityController {
	return &identityController{
		manager: NewIdentityManager(dir),
		cache:   NewIdentityCache(dir, "cache.json"),
	}
}

func (ic *identityController) GetIdentityByValue(id string) *dto.Identity {
	if ic.manager.HasIdentity(id) {
		return ic.manager.GetIdentity(id)
	}

	return nil
}

func (ic *identityController) GetIdentityFromCache() (identity *dto.Identity) {
	id := ic.cache.GetIdentity()

	if id != nil && ic.manager.HasIdentity(string(*id)) {
		return id
	}

	return nil
}

func (ic *identityController) CreateIdentity() (identity *dto.Identity, err error) {
	return ic.manager.CreateNewIdentity("")
}

func (ic *identityController) CacheIdentity(identity *dto.Identity) {
	ic.cache.StoreIdentity(identity)
}
