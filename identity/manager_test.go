package identity

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/stretchr/testify/assert"
)

func newManager(accountValue string) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			AccountsMock: []accounts.Account{
				identityToAccount(accountValue),
			},
		},
	}
}

func newManagerWithError(errorMock error) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			ErrorMock: errorMock,
		},
	}
}

func Test_CreateNewIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")
	identity, err := manager.CreateNewIdentity("")

	assert.NoError(t, err)
	assert.Equal(t, identity, NewIdentity("0x000000000000000000000000000000000000bEEF"))
	assert.Len(t, manager.keystoreManager.Accounts(), 2)
}

func Test_CreateNewIdentityError(t *testing.T) {
	im := newManagerWithError(errors.New("identity create failed"))
	identity, err := im.CreateNewIdentity("")

	assert.EqualError(t, err, "identity create failed")
	assert.Empty(t, identity.Id)
}

func Test_GetIdentities(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		[]Identity{
			NewIdentity("0x000000000000000000000000000000000000000A"),
		},
		manager.GetIdentities(),
	)
}

func Test_GetIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	identity, err := manager.GetIdentity("0x000000000000000000000000000000000000000A")
	assert.Nil(t, err)
	assert.Equal(
		t,
		NewIdentity("0x000000000000000000000000000000000000000A"),
		identity,
	)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000a")
	assert.Nil(t, err)
	assert.Equal(
		t,
		NewIdentity("0x000000000000000000000000000000000000000A"),
		identity,
	)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000B")
	assert.Error(
		t,
		err,
		errors.New("identity not found"),
	)
}

func Test_HasIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000A"))
	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000a"))
	assert.False(t, manager.HasIdentity("0x000000000000000000000000000000000000000B"))
}
