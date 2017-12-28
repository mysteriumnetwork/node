package identity

import (
	"errors"
	"testing"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/stretchr/testify/assert"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func newManager(accountValue string) *identityManager {
	return &identityManager{
		keystoreManager: &keyStoreFake{
			AccountsMock: []accounts.Account{
				identityToAccount(FromAddress(accountValue)),
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
	assert.Equal(t, identity, FromAddress("0x000000000000000000000000000000000000bEEF"))
	assert.Len(t, manager.keystoreManager.Accounts(), 2)
}

func Test_CreateNewIdentityError(t *testing.T) {
	im := newManagerWithError(errors.New("identity create failed"))
	identity, err := im.CreateNewIdentity("")

	assert.EqualError(t, err, "identity create failed")
	assert.Empty(t, identity.Address)
}

func Test_GetIdentities(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		[]Identity{
			FromAddress("0x000000000000000000000000000000000000000A"),
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
		FromAddress("0x000000000000000000000000000000000000000A"),
		identity,
	)

	identity, err = manager.GetIdentity("0x000000000000000000000000000000000000000a")
	assert.Nil(t, err)
	assert.Equal(
		t,
		FromAddress("0x000000000000000000000000000000000000000A"),
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

func Test_IsUnlocked(t *testing.T) {
	ks := getIdentityTestKeystore()

	tests := []struct {
		Address    string
		Passphrase string
	}{
		{
			"0x53a835143c0eF3bBCBFa796D7EB738CA7dd28f68",
			"",
		},
		{
			"0x1e35193c8cadAA15b43B05ae3D882C91F49BB0Aa",
			"test_passphrase",
		},
	}

	// our test identity
	manager := NewIdentityManager(ks)
	for _, test := range tests {
		identity := FromAddress(test.Address)

		assert.False(t, manager.IsUnlocked(identity))

		err := manager.Unlock(identity, test.Passphrase)

		assert.Nil(t, err)
		assert.True(t, manager.IsUnlocked(identity))
	}
}

func Test_UnlockError(t *testing.T) {
	ks := getIdentityTestKeystore()
	testData := struct {
		Address    string
		Passphrase string
	}{
		"0x53a835143c0eF3bBCBFa796D7EB738CA7dd28f68",
		"123",
	}

	// our test identity
	manager := NewIdentityManager(ks)
	identity := FromAddress(testData.Address)

	assert.False(t, manager.IsUnlocked(identity))

	err := manager.Unlock(identity, testData.Passphrase)

	assert.Equal(t, errors.New("could not decrypt key with given passphrase"), err)
	assert.False(t, manager.IsUnlocked(identity))
}

func getIdentityTestKeystore() keystoreInterface {
	ks := keystore.NewKeyStore(
		"test_data",
		keystore.StandardScryptN,
		keystore.StandardScryptP,
	)

	return ks
}
