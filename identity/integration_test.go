package identity

import (
	"testing"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/assert"
	"errors"
)

func getIdentityTestKeystore() keystoreInterface {
	ks := keystore.NewKeyStore(
		"test_data",
		keystore.StandardScryptN,
		keystore.StandardScryptP,
	)

	return ks
}

func TestIsUnlockedWithPasswordAndWithout(t *testing.T) {
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

func TestUnlockWithIncorrectPasswordError(t *testing.T) {
	ks := getIdentityTestKeystore()
	// our test identity
	manager := NewIdentityManager(ks)
	identity := FromAddress("0x53a835143c0eF3bBCBFa796D7EB738CA7dd28f68")

	assert.False(t, manager.IsUnlocked(identity))

	err := manager.Unlock(identity, "123")

	assert.Equal(t, errors.New("could not decrypt key with given passphrase"), err)
	assert.False(t, manager.IsUnlocked(identity))
}