/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package selector

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
)

var log = logconfig.NewLogger()

// IdentityRegistry exposes identity registration method
type IdentityRegistry interface {
	IdentityExists(identity.Identity, identity.Signer) (bool, error)
	RegisterIdentity(identity.Identity, identity.Signer) error
}

type handler struct {
	manager       identity.Manager
	registry      IdentityRegistry
	cache         identity.IdentityCacheInterface
	signerFactory identity.SignerFactory
}

//NewHandler creates new identity handler used by node
func NewHandler(
	manager identity.Manager,
	registry IdentityRegistry,
	cache identity.IdentityCacheInterface,
	signerFactory identity.SignerFactory,
) *handler {
	return &handler{
		manager:       manager,
		registry:      registry,
		cache:         cache,
		signerFactory: signerFactory,
	}
}

func (h *handler) UseOrCreate(address, passphrase string) (id identity.Identity, err error) {
	if len(address) > 0 {
		log.Debug("using existing identity")
		return h.useExisting(address, passphrase)
	}

	identities := h.manager.GetIdentities()
	if len(identities) == 0 {
		log.Debug("creating new identity")
		return h.useNew(passphrase)
	}

	id, err = h.useLast(passphrase)
	if err != nil || !h.manager.HasIdentity(id.Address) {
		log.Debug("using existing identity")
		return h.useExisting(identities[0].Address, passphrase)
	}

	return
}

func (h *handler) useExisting(address, passphrase string) (id identity.Identity, err error) {
	log.Trace("attempting to use existing identity")
	id, err = h.manager.GetIdentity(address)
	if err != nil {
		return id, err
	}

	if err = h.manager.Unlock(id.Address, passphrase); err != nil {
		return id, errors.Wrap(err, "failed to unlock identity")
	}

	registered, err := h.registry.IdentityExists(id, h.signerFactory(id))
	if err != nil {
		return id, errors.Wrap(err, "failed to verify registration status of local identity")
	}
	if !registered {
		log.Info("existing identity is not registered, attempting to register")
		if err = h.registry.RegisterIdentity(id, h.signerFactory(id)); err != nil {
			return id, errors.Wrap(err, "failed to register identity")
		}
	}

	err = h.cache.StoreIdentity(id)
	return id, err
}

func (h *handler) useLast(passphrase string) (identity identity.Identity, err error) {
	log.Trace("attempting to use last identity")
	identity, err = h.cache.GetIdentity()
	if err != nil || !h.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}
	log.Tracef("found identity in cache: %s", identity.Address)

	if err = h.manager.Unlock(identity.Address, passphrase); err != nil {
		return identity, errors.Wrap(err, "failed to unlock identity")
	}
	log.Tracef("unlocked identity: %s", identity.Address)

	return identity, nil
}

func (h *handler) useNew(passphrase string) (id identity.Identity, err error) {
	log.Trace("attempting to use new identity")
	// if all fails, create a new one
	id, err = h.manager.CreateNewIdentity(passphrase)
	if err != nil {
		return id, errors.Wrap(err, "failed to create identity")
	}

	if err = h.manager.Unlock(id.Address, passphrase); err != nil {
		return id, errors.Wrap(err, "failed to unlock identity")
	}

	if err = h.registry.RegisterIdentity(id, h.signerFactory(id)); err != nil {
		return id, errors.Wrap(err, "failed to register identity")
	}

	err = h.cache.StoreIdentity(id)
	return id, errors.Wrap(err, "failed to store identity in cache")
}
