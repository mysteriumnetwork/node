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
	"errors"

	"github.com/mysteriumnetwork/node/identity"
)

// IdentityRegistry exposes identity registration method
type IdentityRegistry interface {
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
		return h.useExisting(address, passphrase)
	}

	identities := h.manager.GetIdentities()
	if len(identities) == 0 {
		id, err = h.useNew(passphrase)

		return id, nil
	}

	id, err = h.useLast(passphrase)
	if err != nil || !h.manager.HasIdentity(id.Address) {
		return h.useExisting(identities[0].Address, passphrase)
	}

	return
}

func (h *handler) useExisting(address, passphrase string) (id identity.Identity, err error) {
	id, err = h.manager.GetIdentity(address)
	if err != nil {
		return
	}

	if err = h.manager.Unlock(id.Address, passphrase); err != nil {
		return
	}

	err = h.cache.StoreIdentity(id)
	return
}

func (h *handler) useLast(passphrase string) (identity identity.Identity, err error) {
	identity, err = h.cache.GetIdentity()
	if err != nil || !h.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}

	if err = h.manager.Unlock(identity.Address, passphrase); err != nil {
		return
	}

	return identity, nil
}

func (h *handler) useNew(passphrase string) (id identity.Identity, err error) {
	// if all fails, create a new one
	id, err = h.manager.CreateNewIdentity(passphrase)
	if err != nil {
		return
	}

	if err = h.manager.Unlock(id.Address, passphrase); err != nil {
		return
	}

	if err = h.registry.RegisterIdentity(id, h.signerFactory(id)); err != nil {
		return
	}

	err = h.cache.StoreIdentity(id)
	return
}
