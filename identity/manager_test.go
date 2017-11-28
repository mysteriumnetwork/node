package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysterium/node/service_discovery/dto"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
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
	identity, err := manager.CreateNewIdentity()

	assert.NoError(t, err)
	assert.Equal(t, *identity, dto.Identity("0x000000000000000000000000000000000000bEEF"))
	assert.Len(t, manager.keystoreManager.Accounts(), 2)
}

func Test_CreateNewIdentityError(t *testing.T) {
	im := newManagerWithError(errors.New("Identity create failed"))
	identity, err := im.CreateNewIdentity()

	assert.EqualError(t, err, "Identity create failed")
	assert.Nil(t, identity)
}

func Test_GetIdentities(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		[]dto.Identity{
			dto.Identity("0x000000000000000000000000000000000000000A"),
		},
		manager.GetIdentities(),
	)
}

func Test_GetIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.Equal(
		t,
		dto.Identity("0x000000000000000000000000000000000000000A"),
		*manager.GetIdentity("0x000000000000000000000000000000000000000A"),
	)
	assert.Equal(
		t,
		dto.Identity("0x000000000000000000000000000000000000000A"),
		*manager.GetIdentity("0x000000000000000000000000000000000000000a"),
	)
	assert.Nil(
		t,
		manager.GetIdentity("0x000000000000000000000000000000000000000B"),
	)
}

func Test_HasIdentity(t *testing.T) {
	manager := newManager("0x000000000000000000000000000000000000000A")

	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000A"))
	assert.True(t, manager.HasIdentity("0x000000000000000000000000000000000000000a"))
	assert.False(t, manager.HasIdentity("0x000000000000000000000000000000000000000B"))
}

func Test_SignMessage(t *testing.T) {
	manager := NewIdentityManager("testdata")
	ids := manager.GetIdentities()
	for _, id := range ids {
		signature, err := manager.SignMessage(id, "message to sign")
		assert.NoError(t, err)
		assert.Len(t, signature, 65)
	}
}
func Test_SignVerifyMessage(t *testing.T) {

	key, err := crypto.GenerateKey()
	assert.NoError(t, err)
	message := []byte("message to sign")

	signature, err := crypto.Sign(signHash(message), key)
	assert.NoError(t, err)

	rpk, err := crypto.Ecrecover(signHash(message), signature)
	assert.NoError(t, err)
	pubKey := crypto.ToECDSAPub(rpk)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	assert.Equal(t, recoveredAddr, crypto.PubkeyToAddress(key.PublicKey))

}