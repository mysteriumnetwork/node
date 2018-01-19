package identity_handler

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
)

type identityHandler struct {
	manager       identity.IdentityManagerInterface
	identityApi   server.Client
	cache         identity.IdentityCacheInterface
	signerFactory identity.SignerFactory
}

//NewNodeIdentityHandler creates new identity handler used by node
func NewNodeIdentityHandler(
	manager identity.IdentityManagerInterface,
	identityApi server.Client,
	cache identity.IdentityCacheInterface,
	signerFactory identity.SignerFactory,
) IdentityHandlerInterface {
	return &identityHandler{
		manager:       manager,
		identityApi:   identityApi,
		cache:         cache,
		signerFactory: signerFactory,
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

func (ih *identityHandler) UseNew(passphrase string) (id identity.Identity, err error) {
	// if all fails, create a new one
	id, err = ih.manager.CreateNewIdentity(passphrase)
	if err != nil {
		return
	}

	if err = ih.identityApi.RegisterIdentity(id, ih.signerFactory(id)); err != nil {
		return
	}

	err = ih.cache.StoreIdentity(id)
	return
}
