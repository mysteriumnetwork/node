package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
	"errors"
)

type identityHandler struct {
	manager *identityManager
	cache   *identityCache
}

func SelectIdentity(dir string, nodeKey string) (id *dto.Identity, err error) {
	handler := NewIdentityHandler(dir)

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

func NewIdentityHandler(dir string) *identityHandler {
	return &identityHandler{
		manager: NewIdentityManager(dir),
		cache:   NewIdentityCache(dir, "cache.json"),
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
