package command_run

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
)

type identityHandler struct {
	manager     identity.IdentityManagerInterface
	identityApi server.Client
	cache       *identity.IdentityCache
}

func NewNodeIdentityHandler(
	manager identity.IdentityManagerInterface,
	identityApi server.Client,
	cacheDir string,
) *identityHandler {
	return &identityHandler{
		manager:     manager,
		identityApi: identityApi,
		cache:       identity.NewIdentityCache(cacheDir, "remember.json"),
	}
}

func (ih *identityHandler) Select(identityAddressWanted string) (id identity.Identity, err error) {
	if len(identityAddressWanted) > 0 {
		return ih.launchExisting(identityAddressWanted)
	}

	if id, err = ih.launchLast(); err == nil {
		return id, err
	}

	return ih.launchNew()
}

func (ih *identityHandler) launchExisting(address string) (id identity.Identity, err error) {
	id, err = ih.manager.GetIdentity(address)
	if err != nil {
		return
	}

	err = ih.cache.StoreIdentity(id)
	return
}

func (ih *identityHandler) launchLast() (identity identity.Identity, err error) {
	identity, err = ih.cache.GetIdentity()
	if err != nil || !ih.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}

	return identity, nil
}

func (ih *identityHandler) launchNew() (id identity.Identity, err error) {
	// if all fails, create a new one
	id, err = ih.manager.CreateNewIdentity("")
	if err != nil {
		return
	}

	if err = ih.identityApi.RegisterIdentity(id); err != nil {
		return
	}

	err = ih.cache.StoreIdentity(id)
	return
}
