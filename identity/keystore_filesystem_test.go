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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const secretMessage = "I like trains. A LOT. Choo CHOO"

func Test_DerivedEncryption(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "derived_encryption")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ks := NewKeystoreFilesystem(dir, true)

	acc, err := ks.NewAccount("")
	assert.NoError(t, err)

	err = ks.Unlock(acc, "")
	assert.NoError(t, err)

	t.Run("Has a derived key for unlocked account", func(t *testing.T) {
		key, err := ks.getDerivedKey(acc.Address)
		assert.NoError(t, err)

		assert.Len(t, key, 32)
	})

	t.Run("Encrypts and decrypts messages with the derived key", func(t *testing.T) {
		encrypted, err := ks.Encrypt(acc.Address, []byte(secretMessage))
		assert.NoError(t, err)
		assert.NotEqual(t, []byte(secretMessage), encrypted)

		decrypted, err := ks.Decrypt(acc.Address, encrypted)
		assert.NoError(t, err)

		assert.Equal(t, secretMessage, string(decrypted))
	})

	t.Run("Errors if message is tampered with", func(t *testing.T) {
		encrypted, err := ks.Encrypt(acc.Address, []byte(secretMessage))
		assert.NoError(t, err)
		assert.NotEqual(t, []byte(secretMessage), encrypted)

		encrypted[len(encrypted)-1] = ^encrypted[len(encrypted)-1]
		_, err = ks.Decrypt(acc.Address, encrypted)
		assert.Error(t, err)
	})

	t.Run("Removes key if account is locked", func(t *testing.T) {
		err := ks.Lock(acc.Address)
		assert.NoError(t, err)

		_, err = ks.getDerivedKey(acc.Address)
		assert.Error(t, err)
	})

	t.Run("Fails to decrypt or encrypt if account is locked", func(t *testing.T) {
		encrypted, err := ks.Encrypt(acc.Address, []byte(secretMessage))
		assert.Error(t, err)

		_, err = ks.Decrypt(acc.Address, encrypted)
		assert.Error(t, err)
	})
}

var result []byte

func Benchmark_DerivedEncryption(b *testing.B) {
	dir, _ := ioutil.TempDir(os.TempDir(), "derived_encryption_bench")
	defer os.RemoveAll(dir)
	ks := NewKeystoreFilesystem(dir, true)
	acc, _ := ks.NewAccount("")
	_ = ks.Unlock(acc, "")

	var r []byte
	for n := 0; n < b.N; n++ {
		encrypted, _ := ks.Encrypt(acc.Address, []byte(secretMessage))
		r = encrypted
	}

	result = r
}

func Benchmark_DerivedDecryption(b *testing.B) {
	dir, _ := ioutil.TempDir(os.TempDir(), "derived_encryption_bench")
	defer os.RemoveAll(dir)
	ks := NewKeystoreFilesystem(dir, true)
	acc, _ := ks.NewAccount("")
	_ = ks.Unlock(acc, "")
	encrypted, _ := ks.Encrypt(acc.Address, []byte(secretMessage))

	var r []byte
	for n := 0; n < b.N; n++ {
		decrypted, _ := ks.Decrypt(acc.Address, encrypted)
		r = decrypted
	}

	result = r
}
