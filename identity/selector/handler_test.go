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
var chainID int64 = 1

func TestUseOrCreateSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	id, err := handler.UseOrCreate(existingIdentity.Address, "pass", chainID)
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, existingIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
}

func TestUseOrCreateFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	_, err := handler.UseOrCreate(existingIdentity.Address, "pass", chainID)
	assert.Error(t, err)

	assert.Equal(t, existingIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
}

func TestUseOrCreateReturnsFirstIdentity(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity, newIdentity}, newIdentity)
	cache := identity.NewIdentityCacheFake()
	_ = cache.StoreIdentity(existingIdentity)

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	id, err := handler.UseOrCreate("", "pass", chainID)
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
}

func TestUseOrCreateFailsWhenIdentityNotFound(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)
	_, err := handler.UseOrCreate("does-not-exist", "pass", chainID)
	assert.NotNil(t, err)
}

func TestUseFailsWhenIdentityNotFound(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	_, err := handler.UseOrCreate("does-not-exist", "pass", chainID)
	assert.NotNil(t, err)
}

func TestUseLastSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	_ = cache.StoreIdentity(fakeIdentity)

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	id, err := handler.useLast("pass", chainID)
	assert.Equal(t, fakeIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, "abc", identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
}

func TestUseLastFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	registry := &mockRegistry{}
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	_ = cache.StoreIdentity(fakeIdentity)

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	_, err := handler.useLast("pass", chainID)
	assert.Error(t, err)

	assert.Equal(t, "", registry.registeredIdentity.Address)

	assert.Equal(t, "abc", identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
}

func TestUseNewSucceeds(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	id, err := handler.useNew("pass", chainID)
	assert.Equal(t, newIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, newIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
}

func TestUseNewFailsWhenUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.MarkUnlockToFail()
	cache := identity.NewIdentityCacheFake()

	handler := NewHandler(identityManager, cache, fakeSignerFactory)

	_, err := handler.useNew("pass", chainID)
	assert.Error(t, err)

	assert.Equal(t, newIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
	assert.Equal(t, chainID, identityManager.LastUnlockChainID)
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
