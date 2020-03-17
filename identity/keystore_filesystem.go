/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"io"
	"sync"

	"github.com/awnumar/memguard"
	"github.com/ethereum/go-ethereum/accounts"
	ethKs "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/hkdf"
)

// NewKeystoreFilesystem create new keystore, which keeps keys in filesystem.
func NewKeystoreFilesystem(directory string, lightweight bool) *Keystore {
	var ks *ethKs.KeyStore
	if lightweight {
		log.Debug().Msg("Using lightweight keystore")
		ks = ethKs.NewKeyStore(directory, ethKs.LightScryptN, ethKs.LightScryptP)
	} else {
		log.Debug().Msg("using heavyweight keystore")
		ks = ethKs.NewKeyStore(directory, ethKs.StandardScryptN, ethKs.StandardScryptP)
	}

	return &Keystore{
		ethKeystore: ks,
		derivedKeys: make(map[common.Address]*memguard.Enclave),
	}
}

// Keystore handles everything that's related to eth accounts.
type Keystore struct {
	ethKeystore *ethKs.KeyStore

	derivedKeys    map[common.Address]*memguard.Enclave
	derivedKeyLock sync.Mutex
}

// Accounts returns all accounts.
func (ks *Keystore) Accounts() []accounts.Account {
	return ks.ethKeystore.Accounts()
}

// NewAccount creates a new account.
func (ks *Keystore) NewAccount(passphrase string) (accounts.Account, error) {
	return ks.ethKeystore.NewAccount(passphrase)
}

// Find finds an account.
func (ks *Keystore) Find(a accounts.Account) (accounts.Account, error) {
	return ks.ethKeystore.Find(a)
}

// Unlock unlocks an account.
func (ks *Keystore) Unlock(a accounts.Account, passphrase string) error {
	err := ks.ethKeystore.Unlock(a, passphrase)
	if err != nil {
		return err
	}

	derived, err := ks.deriveKey(a, passphrase)
	if err != nil {
		return err
	}

	ks.rememberDerivedKey(a.Address, derived)
	return nil
}

// Lock locks an account.
func (ks *Keystore) Lock(addr common.Address) error {
	defer ks.forgetDerivedKey(addr)
	return ks.ethKeystore.Lock(addr)
}

func (ks *Keystore) rememberDerivedKey(a common.Address, key []byte) {
	ks.derivedKeyLock.Lock()
	defer ks.derivedKeyLock.Unlock()
	enclave := memguard.NewEnclave(key)
	ks.derivedKeys[a] = enclave
}

func (ks *Keystore) forgetDerivedKey(a common.Address) {
	ks.derivedKeyLock.Lock()
	defer ks.derivedKeyLock.Unlock()
	delete(ks.derivedKeys, a)
}

func (ks *Keystore) getDerivedKey(a common.Address) ([]byte, error) {
	ks.derivedKeyLock.Lock()
	defer ks.derivedKeyLock.Unlock()

	if _, ok := ks.derivedKeys[a]; !ok {
		return nil, errors.New("no key found")
	}

	enclave := ks.derivedKeys[a]
	buffer, err := enclave.Open()
	if err != nil {
		return nil, err
	}
	defer buffer.Destroy()

	bytes := buffer.Bytes()

	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	return copied, nil
}

func (ks *Keystore) deriveKey(a accounts.Account, passphrase string) ([]byte, error) {
	kjson, err := ks.ethKeystore.Export(a, passphrase, passphrase)
	if err != nil {
		return nil, err
	}

	k, err := ethKs.DecryptKey(kjson, passphrase)
	if err != nil {
		return nil, err
	}
	defer memguard.WipeBytes(k.PrivateKey.D.Bytes())

	hashFunc := sha512.New
	hkdfDeriver := hkdf.New(hashFunc, k.PrivateKey.D.Bytes(), nil, nil)
	key := make([]byte, 32)
	_, err = io.ReadFull(hkdfDeriver, key)
	return key, err
}

// Encrypt takes a derived key for the given address and encrypts the plaintext.
func (ks *Keystore) Encrypt(addr common.Address, plaintext []byte) ([]byte, error) {
	key, err := ks.getDerivedKey(addr)
	if err != nil {
		return nil, err
	}
	defer memguard.WipeBytes(key)

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt takes a derived key for the given address and decrypts the encrypted message.
func (ks *Keystore) Decrypt(addr common.Address, encrypted []byte) ([]byte, error) {
	key, err := ks.getDerivedKey(addr)
	if err != nil {
		return nil, err
	}
	defer memguard.WipeBytes(key)

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, encrypted := encrypted[:nonceSize], encrypted[nonceSize:]
	return gcm.Open(nil, nonce, encrypted, nil)
}

// SignHash signs the given hash.
func (ks *Keystore) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	return ks.ethKeystore.SignHash(a, hash)
}
