package identity

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
)

type handler struct {
	manager       identity.IdentityManagerInterface
	identityApi   server.Client
	cache         identity.IdentityCacheInterface
	signerFactory identity.SignerFactory
}

//NewHandler creates new identity handler used by node
func NewHandler(
	manager identity.IdentityManagerInterface,
	identityApi server.Client,
	cache identity.IdentityCacheInterface,
	signerFactory identity.SignerFactory,
) HandlerInterface {
	return &handler{
		manager:       manager,
		identityApi:   identityApi,
		cache:         cache,
		signerFactory: signerFactory,
	}
}

func (h *handler) UseExisting(address string) (id identity.Identity, err error) {
	id, err = h.manager.GetIdentity(address)
	if err != nil {
		return
	}

	err = h.cache.StoreIdentity(id)
	return
}

func (h *handler) UseLast() (identity identity.Identity, err error) {
	identity, err = h.cache.GetIdentity()
	if err != nil || !h.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}

	return identity, nil
}

func (h *handler) UseNew(passphrase string) (id identity.Identity, err error) {
	// if all fails, create a new one
	id, err = h.manager.CreateNewIdentity(passphrase)
	if err != nil {
		return
	}

	if err = h.identityApi.RegisterIdentity(id, h.signerFactory(id)); err != nil {
		return
	}

	err = h.cache.StoreIdentity(id)
	return
}
