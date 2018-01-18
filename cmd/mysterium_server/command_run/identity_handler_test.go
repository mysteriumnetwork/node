package command_run

import (
	"testing"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/stretchr/testify/assert"
)

var fakeSignerFactory = func(id identity.Identity) identity.Signer {
	return &fakeSigner{}
}
var existingIdentity = identity.Identity{"existing"}
var newIdentity = identity.Identity{"new"}

func Test_identityHandler_UseExisting(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseExisting(existingIdentity.Address, "pass")
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
	assert.Equal(t, "", client.RegisteredIdentity.Address)

	assert.Equal(t, existingIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func Test_identityHandler_UseExistingNotFound(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	_, err := handler.UseExisting("does-not-exist", "")
	assert.NotNil(t, err)
}

func Test_identityHandler_UseExistingUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.UnlockFails = true
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	_, err := handler.UseExisting(existingIdentity.Address, "abc")
	assert.NotNil(t, err)
}

func Test_identityHandler_UseLast(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	cache.StoreIdentity(fakeIdentity)

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseLast("pass")
	assert.Equal(t, fakeIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, "", client.RegisteredIdentity.Address)

	assert.Equal(t, "abc", identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func Test_identityHandler_UseLastUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.UnlockFails = true
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	cache.StoreIdentity(fakeIdentity)

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	_, err := handler.UseLast("pass")
	assert.NotNil(t, err)
}

func Test_identityHandler_UseNew(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseNew("pass")
	assert.Equal(t, newIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, newIdentity, client.RegisteredIdentity)

	assert.Equal(t, newIdentity.Address, identityManager.LastUnlockAddress)
	assert.Equal(t, "pass", identityManager.LastUnlockPassphrase)
}

func Test_identityHandler_UseNewUnlockFails(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)
	identityManager.UnlockFails = true
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	_, err := handler.UseNew("pass")
	assert.NotNil(t, err)
}

type fakeSigner struct {
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	return identity.SignatureBase64("deadbeef"), nil
}
