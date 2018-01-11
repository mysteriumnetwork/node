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
var identityManager = identity.NewIdentityManagerFake([]identity.Identity{existingIdentity}, newIdentity)

func Test_identityHandler_UseExisting(t *testing.T) {
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseExisting(existingIdentity.Address)
	assert.Equal(t, existingIdentity, id)
	assert.Nil(t, err)
	assert.Equal(t, "", client.RegisteredIdentity.Address)
}

func Test_identityHandler_UseExistingNotFound(t *testing.T) {
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	_, err := handler.UseExisting("does-not-exist")
	assert.NotNil(t, err)
}

func Test_identityHandler_UseLast(t *testing.T) {
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	cache.StoreIdentity(fakeIdentity)

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseLast()
	assert.Equal(t, fakeIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, "", client.RegisteredIdentity.Address)
}

func Test_identityHandler_UseNew(t *testing.T) {
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache, fakeSignerFactory)

	id, err := handler.UseNew()
	assert.Equal(t, newIdentity, id)
	assert.Nil(t, err)

	assert.Equal(t, newIdentity, client.RegisteredIdentity)
}

type fakeSigner struct {
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	return identity.SignatureBase64Decode("deadbeef"), nil
}
