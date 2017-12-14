package identity

import (
	"errors"
	"github.com/mysterium/node/service_discovery/dto"
)

type identityHandler struct {
	manager IdentityManagerInterface
	cache   *identityCache
}

func SelectIdentity(keystore keystoreInterface, cacheDir, nodeKey string) (id *dto.Identity, err error) {
	handler := NewIdentityHandler(keystore, cacheDir)

	// validate and return user provided identity
	if len(nodeKey) > 0 {
		id = handler.GetIdentityByValue(nodeKey)
		if id == nil {
			return id, errors.New("identity doesn't exist")
		}
		handler.CacheIdentity(id)
		return
	}

	// try cache
	id = handler.GetIdentityFromCache()
	if id != nil {
		handler.CacheIdentity(id)
		return
	}

	return
}

func CreateIdentity(dir string) (id *dto.Identity, err error) {
	handler := NewIdentityHandler(dir)

	// if all fails, create a new one
	id, err = handler.CreateIdentity()
	if err != nil {
		return id, err
	}

	handler.CacheIdentity(id)

	return
}

func NewIdentityHandler(keystore keystoreInterface, cacheDir string) *identityHandler {
	return &identityHandler{
		manager: NewIdentityManager(keystore),
		cache:   NewIdentityCache(cacheDir, "cache.json"),
	}
}

func (ih *identityHandler) GetIdentityByValue(id string) *dto.Identity {
	if ih.manager.HasIdentity(id) {
		return ih.manager.GetIdentity(id)
	}

	return nil
}

func (ih *identityHandler) GetIdentityFromCache() (identity *dto.Identity) {
	id := ih.cache.GetIdentity()

	if id != nil && ih.manager.HasIdentity(string(*id)) {
		return id
	}

	return nil
}

func (ih *identityHandler) CreateIdentity() (identity *dto.Identity, err error) {
	return ih.manager.CreateNewIdentity("")
}

func (ih *identityHandler) CacheIdentity(identity *dto.Identity) {
	ih.cache.StoreIdentity(identity)
}
