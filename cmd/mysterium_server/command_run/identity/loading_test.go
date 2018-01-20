package identity

import (
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"testing"
)

var fakeSuccessIdentitySelector = func() (identity.Identity, error) {
	return identity.Identity{Address: "fake"}, nil
}

var fakeFailureIdentitySelector = func() (identity.Identity, error) {
	return identity.Identity{}, errors.New("selection failed")
}

func Test_LoadIdentity(t *testing.T) {
	fakeIdentityManager := identity.NewIdentityManagerFake(nil, identity.Identity{})
	id, err := LoadIdentity(fakeSuccessIdentitySelector, fakeIdentityManager, "pass")
	assert.Equal(t, "fake", id.Address)
	assert.Nil(t, err)

	assert.Equal(t, "fake", fakeIdentityManager.LastUnlockAddress)
	assert.Equal(t, "pass", fakeIdentityManager.LastUnlockPassphrase)
}

func Test_LoadIdentitySelectionFails(t *testing.T) {
	fakeIdentityManager := identity.NewIdentityManagerFake(nil, identity.Identity{})
	_, err := LoadIdentity(fakeFailureIdentitySelector, fakeIdentityManager, "pass")
	assert.NotNil(t, err)

	assert.Equal(t, "", fakeIdentityManager.LastUnlockAddress)
	assert.Equal(t, "", fakeIdentityManager.LastUnlockPassphrase)
}

func Test_LoadIdentityUnlockFails(t *testing.T) {
	fakeIdentityManager := identity.NewIdentityManagerFake(nil, identity.Identity{})
	fakeIdentityManager.MarkUnlockToFail()
	_, err := LoadIdentity(fakeSuccessIdentitySelector, fakeIdentityManager, "pass")
	assert.NotNil(t, err)

	assert.Equal(t, "fake", fakeIdentityManager.LastUnlockAddress)
	assert.Equal(t, "pass", fakeIdentityManager.LastUnlockPassphrase)
}

func Test_SelectIdentityExisting(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := SelectIdentity(identityHandler, "existing", "")
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_SelectIdentityLast(t *testing.T) {
	identityHandler := &handlerFake{LastAddress: "last"}
	id, err := SelectIdentity(identityHandler, "", "")
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_SelectIdentityNew(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := SelectIdentity(identityHandler, "", "")
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}
