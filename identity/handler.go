package identity

import (
	"errors"
	"github.com/mysterium/node/service_discovery/dto"
)

type identityHandler struct {
	manager IdentityManagerInterface
	cache   *identityCache
}

func NewNodeIdentityHandler(keystore keystoreInterface, cacheDir string) *identityHandler {
	return &identityHandler{
		manager: NewIdentityManager(keystore),
		cache:   NewIdentityCache(cacheDir, "remember.json"),
	}
}

func (ih *identityHandler) Select(nodeKey string) (id dto.Identity, err error) {
	if len(nodeKey) > 0 {
		id, err = ih.getIdentityByValue(nodeKey)
		if err != nil {
			return id, err
		}

		ih.cacheIdentity(id)
		return
	}

	id, err = ih.getIdentityFromCache()
	if err != nil {
		return
	}

	ih.cacheIdentity(id)

	return
}

func (ih *identityHandler) Create() (id dto.Identity, err error) {
	// if all fails, create a new one
	id, err = ih.createIdentity()
	if err != nil {
		return id, err
	}

	ih.cacheIdentity(id)

	return
}

func (ih *identityHandler) getIdentityByValue(id string) (dto.Identity, error) {
	if ih.manager.HasIdentity(id) {
		return ih.manager.GetIdentity(id), nil
	}

	return dto.Identity(""), errors.New("identity not found")
}

func (ih *identityHandler) getIdentityFromCache() (identity dto.Identity, err error) {
	id := ih.cache.GetIdentity()

	if len(id) > 0 && ih.manager.HasIdentity(string(id)) {
		return id, nil
	}

	return identity, errors.New("identity not found in cache")
}

func (ih *identityHandler) createIdentity() (identity dto.Identity, err error) {
	return ih.manager.CreateNewIdentity("")
}

func (ih *identityHandler) cacheIdentity(identity dto.Identity) {
	ih.cache.StoreIdentity(identity)
}
