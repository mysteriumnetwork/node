/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package identity

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	ethKs "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const secretMessage = "I like trains. A LOT. Choo CHOO"

var (
	encryptionAddress = common.HexToAddress("53a835143c0ef3bbcbfa796d7eb738ca7dd28f68")
	encryptionAccount = accounts.Account{Address: encryptionAddress}
	encryptionKey, _  = crypto.HexToECDSA("6f88637b68ee88816e73f663aef709d7009836c98ae91ef31e3dfac7be3a1657")
)

func Test_DerivedEncryption(t *testing.T) {
	ks := NewKeystoreFilesystem("", &ethKeystoreMock{account: encryptionAccount})
	ks.loadKey = func(addr common.Address, filename, auth string) (*ethKs.Key, error) {
		return &ethKs.Key{Address: addr, PrivateKey: encryptionKey}, nil
	}

	t.Run("Fails to decrypt or encrypt if account is locked", func(t *testing.T) {
		encrypted, err := ks.Encrypt(encryptionAddress, []byte(secretMessage))
		assert.Error(t, err)

		_, err = ks.Decrypt(encryptionAddress, encrypted)
		assert.Error(t, err)
	})

	err := ks.Unlock(encryptionAccount, "")
	assert.NoError(t, err)

	t.Run("Encrypts and decrypts messages with the derived key", func(t *testing.T) {
		encrypted, err := ks.Encrypt(encryptionAddress, []byte(secretMessage))
		assert.NoError(t, err)
		assert.NotEqual(t, []byte(secretMessage), encrypted)

		decrypted, err := ks.Decrypt(encryptionAddress, encrypted)
		assert.NoError(t, err)

		assert.Equal(t, secretMessage, string(decrypted))
	})

	t.Run("Errors if message is tampered with", func(t *testing.T) {
		encrypted, err := ks.Encrypt(encryptionAddress, []byte(secretMessage))
		assert.NoError(t, err)
		assert.NotEqual(t, []byte(secretMessage), encrypted)

		encrypted[1] = 0x1
		_, err = ks.Decrypt(encryptionAddress, encrypted)
		assert.Error(t, err)
	})
}

var result []byte

func Benchmark_DerivedEncryption(b *testing.B) {
	ks := NewKeystoreFilesystem("", &ethKeystoreMock{account: encryptionAccount})
	ks.loadKey = func(addr common.Address, filename, auth string) (*ethKs.Key, error) {
		return &ethKs.Key{}, nil
	}
	_ = ks.Unlock(encryptionAccount, "")

	var r []byte
	for n := 0; n < b.N; n++ {
		encrypted, _ := ks.Encrypt(encryptionAddress, []byte(secretMessage))
		r = encrypted
	}

	result = r
}

func Benchmark_DerivedDecryption(b *testing.B) {
	ks := NewKeystoreFilesystem("", &ethKeystoreMock{account: encryptionAccount})
	ks.loadKey = func(addr common.Address, filename, auth string) (*ethKs.Key, error) {
		return &ethKs.Key{}, nil
	}
	_ = ks.Unlock(encryptionAccount, "")
	encrypted, _ := ks.Encrypt(encryptionAddress, []byte(secretMessage))

	var r []byte
	for n := 0; n < b.N; n++ {
		decrypted, _ := ks.Decrypt(encryptionAddress, encrypted)
		r = decrypted
	}

	result = r
}

type ethKeystoreMock struct {
	account  accounts.Account
	unlocked bool
}

func (ekm *ethKeystoreMock) Unlock(a accounts.Account, passphrase string) error {
	ekm.unlocked = true
	return nil
}

func (ekm *ethKeystoreMock) Delete(a accounts.Account, passphrase string) error {
	if a.Address == ekm.account.Address {
		ekm.account = accounts.Account{}
	}

	return nil
}

func (ekm *ethKeystoreMock) Export(a accounts.Account, passphrase, newPassphrase string) ([]byte, error) {
	return []byte("exported"), nil
}

func (ekm *ethKeystoreMock) Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error) {
	return ekm.account, nil
}

func (ekm *ethKeystoreMock) Accounts() []accounts.Account {
	return []accounts.Account{ekm.account}
}

func (ekm *ethKeystoreMock) Find(a accounts.Account) (accounts.Account, error) {
	if ekm.account.Address == a.Address {
		return ekm.account, nil
	}
	return accounts.Account{}, errors.New("not found")
}

func (ekm *ethKeystoreMock) NewAccount(passphrase string) (accounts.Account, error) {
	return accounts.Account{}, errors.New("not implemented yet")
}
