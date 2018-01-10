package command_run

import (
	"testing"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/stretchr/testify/assert"
)

func Test_identityHandler_UseExisting(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake()
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache)

	id, err := handler.UseExisting("address")
	assert.Equal(t, identityManager.FakeIdentity1, id)
	assert.Nil(t, err)
	assert.Equal(t, "", client.RegisteredIdentity.Address)
}

func Test_identityHandler_UseLast(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake()
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	fakeIdentity := identity.FromAddress("abc")
	cache.StoreIdentity(fakeIdentity)

	handler := NewNodeIdentityHandler(identityManager, client, cache)

	id, err := handler.UseLast()
	assert.Equal(t, fakeIdentity, id)
	assert.Nil(t, err)
	assert.Equal(t, "", client.RegisteredIdentity.Address)
}

func Test_identityHandler_UseNew(t *testing.T) {
	identityManager := identity.NewIdentityManagerFake()
	client := server.NewClientFake()
	cache := identity.NewIdentityCacheFake()

	handler := NewNodeIdentityHandler(identityManager, client, cache)

	id, err := handler.UseNew()
	assert.Equal(t, identityManager.FakeIdentity2, client.RegisteredIdentity)
	assert.Equal(t, identityManager.FakeIdentity2, id)
	assert.Nil(t, err)
}
