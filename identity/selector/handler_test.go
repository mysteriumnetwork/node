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
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var fakeSignerFactory = func(id identity.Identity) identity.Signer {
	return &fakeSigner{}
}
var existingIdentity = identity.Identity{"existing"}
var newIdentity = identity.Identity{"new"}

func TestUseOrCreateSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	id, err := handler.UseOrCreate(existingIdentity.Address, "pass")
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, existingIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func TestUUseOrCreateFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	_, err := handler.UseOrCreate(existingIdentity.Address, "pass")
	assert.Error(t, err)

	assert.Equal(t, existingIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func TestUseOrCreateReturnsFirstIdentity(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity, newIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()
	_ = cache.StoreIdentity(existingIdentity)

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	id, err := handler.UseOrCreate("", "pass")
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
}

func TestUseOrCreateFailsWhenIdentityNotFound(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	_, err := handler.UseOrCreate("does-not-exist", "pass")
	assert.NotNil(t, err)
}

type fakeSigner struct {
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	return identity.SignatureBase64("deadbeef"), nil
}

type mockRegistry struct {
	registeredIdentity identity.Identity
}

func (mr *mockRegistry) RegisterIdentity(ID identity.Identity, signer identity.Signer) error {
	mr.registeredIdentity = ID
	return nil
}

var _ IdentityRegistry = &mockRegistry{}
