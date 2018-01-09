package command_run

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
)

const nodeIdentityPassword = ""

type identityHandler struct {
	manager     identity.IdentityManagerInterface
	identityApi server.Client
	cache       identity.IdentityCacheInterface
}

func NewNodeIdentityHandler(
	manager identity.IdentityManagerInterface,
	identityApi server.Client,
	cache identity.IdentityCacheInterface,
) *identityHandler {
	return &identityHandler{
		manager:     manager,
		identityApi: identityApi,
		cache:       cache,
	}
}

func (ih *identityHandler) UseExisting(address string) (id identity.Identity, err error) {
	id, err = ih.manager.GetIdentity(address)
	if err != nil {
		return
	}

	err = ih.cache.StoreIdentity(id)
	return
}

func (ih *identityHandler) UseLast() (identity identity.Identity, err error) {
	identity, err = ih.cache.GetIdentity()
	if err != nil || !ih.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}

	return identity, nil
}

func (ih *identityHandler) UseNew() (id identity.Identity, err error) {
	// if all fails, create a new one
	id, err = ih.manager.CreateNewIdentity(nodeIdentityPassword)
	if err != nil {
		return
	}

	if err = ih.identityApi.RegisterIdentity(id); err != nil {
		return
	}

	err = ih.cache.StoreIdentity(id)
	return
}
