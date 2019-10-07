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
var existingIdentity = identity.Identity{Address: "existing"}
var newIdentity = identity.Identity{Address: "new"}

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

func TestUseOrCreateFailsWhenUnlockFails(t *testing.T) {
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

func TestUseFailsWhenIdentityNotFound(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	_, err := handler.UseOrCreate("does-not-exist", "pass")
	assert.NotNil(t, err)
}

func TestUseLastSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	_ = cache.StoreIdentity(fakeIdentity)

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	id, err := handler.useLast("pass")
	assert.Equal(t, fakeIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, "abc", identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func TestUseLastFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	_ = cache.StoreIdentity(fakeIdentity)

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	_, err := handler.useLast("pass")
	assert.Error(t, err)

	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, "abc", identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func TestUseNewSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	id, err := handler.useNew("pass")
	assert.Equal(t, newIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, newIdentity, registry.registeredIdentity)

	assert.Equal(t, newIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func TestUseNewFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, registry, cache, fakeSignerFactory)

	_, err := handler.useNew("pass")
	assert.Error(t, err)

	assert.Equal(t, newIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

type fakeSigner struct {
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	return identity.SignatureBase64("deadbeef"), nil
}

type mockRegistry struct {
	registeredIdentity identity.Identity
}

func (mr *mockRegistry) IdentityExists(identity.Identity, identity.Signer) (bool, error) {
	return true, nil
}

func (mr *mockRegistry) RegisterIdentity(ID identity.Identity, signer identity.Signer) error {
	mr.registeredIdentity = ID
	return nil
}

var _ IdentityRegistry = &mockRegistry{}
