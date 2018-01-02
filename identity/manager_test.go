package identity

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newManager(accountValue string) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			AccountsMock: []accounts.Account{
				addressToAccount(accountValue),
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

func TestManager_CreateNewIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")
	identity, err := manager.CreateNewIdentity("")

	assert.NoError(t, err)
	assert.Equal(t, identity, Identity{"0x000000000000000000000000000000000000beef"})
	assert.Len(t, manager.keystoreManager.Accounts(), 2)
}

func TestManager_CreateNewIdentityError(t *testing.T) {
	im := newManagerWithError(errors.New("identity create failed"))
	identity, err := im.CreateNewIdentity("")

	assert.EqualError(t, err, "identity create failed")
	assert.Empty(t, identity.Address)
}

func TestManager_GetIdentities(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		[]Identity{
			{"0x000000000000000000000000000000000000000a"},
		},
		manager.GetIdentities(),
	)
}

func TestManager_GetIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	identity, err := manager.GetIdentity("0x000000000000000000000000000000000000000A")
	assert.NoError(t, err)
	assert.Exactly(t, Identity{"0x000000000000000000000000000000000000000a"}, identity)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000a")
	assert.NoError(t, err)
	assert.Exactly(t, Identity{"0x000000000000000000000000000000000000000a"}, identity)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000B")
	assert.EqualError(t, err, "identity not found")
	assert.Exactly(t, Identity{}, identity)
}

func TestManager_HasIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000A"))
	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000a"))
	assert.False(t, manager.HasIdentity("0x000000000000000000000000000000000000000B"))
}
