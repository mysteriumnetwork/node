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
	"fmt"
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type handler struct {
	mu            sync.Mutex
	manager       identity.Manager
	cache         identity.IdentityCacheInterface
	signerFactory identity.SignerFactory
}

// NewHandler creates new identity handler used by node
func NewHandler(
	manager identity.Manager,
	cache identity.IdentityCacheInterface,
	signerFactory identity.SignerFactory,
) *handler {
	return &handler{
		manager:       manager,
		cache:         cache,
		signerFactory: signerFactory,
	}
}

func (h *handler) UseOrCreate(address, passphrase string, chainID int64) (id identity.Identity, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(address) > 0 {
		log.Debug().Msg("Using existing identity")
		return h.useExisting(address, passphrase, chainID)
	}

	identities := h.manager.GetIdentities()
	if len(identities) == 0 {
		log.Debug().Msg("Creating new identity")
		return h.useNew(passphrase, chainID)
	}

	id, err = h.useLast(passphrase, chainID)
	if err != nil || !h.manager.HasIdentity(id.Address) {
		log.Debug().Msg("Using existing identity")
		return h.useExisting(identities[0].Address, passphrase, chainID)
	}

	return
}

func (h *handler) SetDefault(address string) error {
	id, err := h.manager.GetIdentity(address)
	if err != nil {
		return err
	}

	return h.cache.StoreIdentity(id)
}

func (h *handler) useExisting(address, passphrase string, chainID int64) (id identity.Identity, err error) {
	log.Debug().Msg("Attempting to use existing identity")
	id, err = h.manager.GetIdentity(address)
	if err != nil {
		return id, err
	}

	if err = h.manager.Unlock(chainID, id.Address, passphrase); err != nil {
		return id, fmt.Errorf("failed to unlock identity: %w", err)
	}

	err = h.cache.StoreIdentity(id)
	return id, err
}

func (h *handler) useLast(passphrase string, chainID int64) (identity identity.Identity, err error) {
	log.Debug().Msg("Attempting to use last identity")
	identity, err = h.cache.GetIdentity()
	if err != nil || !h.manager.HasIdentity(identity.Address) {
		return identity, errors.New("identity not found in cache")
	}
	log.Debug().Msg("Found identity in cache: " + identity.Address)

	if err = h.manager.Unlock(chainID, identity.Address, passphrase); err != nil {
		return identity, errors.Wrap(err, "failed to unlock identity")
	}
	log.Debug().Msg("Unlocked identity: " + identity.Address)

	return identity, nil
}

func (h *handler) useNew(passphrase string, chainID int64) (id identity.Identity, err error) {
	log.Debug().Msg("Attempting to use new identity")
	// if all fails, create a new one
	id, err = h.manager.CreateNewIdentity(passphrase)
	if err != nil {
		return id, errors.Wrap(err, "failed to create identity")
	}

	if err = h.manager.Unlock(chainID, id.Address, passphrase); err != nil {
		return id, errors.Wrap(err, "failed to unlock identity")
	}

	err = h.cache.StoreIdentity(id)
	return id, errors.Wrap(err, "failed to store identity in cache")
}
