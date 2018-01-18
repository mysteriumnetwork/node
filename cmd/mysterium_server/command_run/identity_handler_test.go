package command_run

import (
	"testing"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var fakeSignerFactory = func(id identity.Identity) identity.Signer {
	return &fakeSigner{}
}
var existingIdentity = identity.Identity{"existing"}
var newIdentity = identity.Identity{"new"}

type identityHandlerFake struct {
	LastAddress string
}

func (ihf *identityHandlerFake) UseExisting(address, passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: address}, nil
}

func (ihf *identityHandlerFake) UseLast(passphrase string) (id identity.Identity, err error) {
	if ihf.LastAddress != "" {
		id = identity.Identity{Address: ihf.LastAddress}
	} else {
		err = errors.New("No last identity")
	}
	return
}

func (ihf *identityHandlerFake) UseNew(passphrase string) (identity.Identity, error) {
	return identity.Identity{Address: "new"}, nil
}

func Test_LoadIdentityExisting(t *testing.T) {
	identityHandler := &identityHandlerFake{}
	id, err := LoadIdentity(identityHandler, "existing", "")
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityLast(t *testing.T) {
	identityHandler := &identityHandlerFake{LastAddress: "last"}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityNew(t *testing.T) {
	identityHandler := &identityHandlerFake{}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}

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
