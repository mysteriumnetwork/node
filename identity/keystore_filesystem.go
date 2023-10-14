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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	ethKs "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/hkdf"
)

type ethKeystore interface {
	Delete(a accounts.Account, passphrase string) error
	Accounts() []accounts.Account
	NewAccount(passphrase string) (accounts.Account, error)
	Find(a accounts.Account) (accounts.Account, error)
	Export(a accounts.Account, passphrase, newPassphrase string) ([]byte, error)
	Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error)
}

// NewKeystoreFilesystem create new keystore, which keeps keys in filesystem.
func NewKeystoreFilesystem(directory string, ks ethKeystore) *Keystore {
	return &Keystore{
		ethKeystore: ks,
		loadKey:     loadStoredKey,
		unlocked:    make(map[common.Address]*unlocked),
	}
}

// Keystore handles everything that's related to eth accounts.
type Keystore struct {
	ethKeystore
	loadKey func(addr common.Address, filename, auth string) (*ethKs.Key, error)

	unlocked map[common.Address]*unlocked // Currently unlocked account (decrypted private keys)
	mu       sync.RWMutex
}

// Unlock unlocks the given account indefinitely.
func (ks *Keystore) Unlock(a accounts.Account, passphrase string) error {
	return ks.TimedUnlock(a, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (ks *Keystore) Lock(addr common.Address) error {
	ks.mu.Lock()
	if unl, found := ks.unlocked[addr]; found {
		ks.mu.Unlock()
		ks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		ks.mu.Unlock()
	}
	return nil
}

// TimedUnlock unlocks the given account with the passphrase. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (ks *Keystore) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()
	u, found := ks.unlocked[a.Address]
	if found {
		if u.abort == nil {
			// The address was unlocked indefinitely, so unlocking
			// it with a timeout would be confusing.
			zeroKey(key.PrivateKey)
			return nil
		}
		// Terminate the expire goroutine and replace it below.
		close(u.abort)
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go ks.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	ks.unlocked[a.Address] = u
	return nil
}

func (ks *Keystore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *ethKs.Key, error) {
	a, err := ks.ethKeystore.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.loadKey(a.Address, a.URL.Path, auth)
	return a, key, err
}

func (ks *Keystore) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if ks.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(ks.unlocked, addr)
		}
		ks.mu.Unlock()
	}
}

// Encrypt takes a derived key for the given address and encrypts the plaintext.
func (ks *Keystore) Encrypt(addr common.Address, plaintext []byte) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	key, found := ks.unlocked[addr]
	if !found {
		return nil, ethKs.ErrLocked
	}

	keyDerived, err := key.deriveKey()
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(keyDerived)
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
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	key, found := ks.unlocked[addr]
	if !found {
		return nil, ethKs.ErrLocked
	}

	keyDerived, err := key.deriveKey()
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(keyDerived)
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

// SignHash calculates a ECDSA signature for the given hash. The produced
// signature is in the [R || S || V] format where V is 0 or 1.
func (ks *Keystore) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ethKs.ErrLocked
	}
	// Sign the hash using plain ECDSA operations
	return crypto.Sign(hash, unlockedKey.PrivateKey)
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

type unlocked struct {
	*ethKs.Key
	abort chan struct{}
}

func (u *unlocked) deriveKey() ([]byte, error) {
	hashFunc := sha512.New
	hkdfDerived := hkdf.New(hashFunc, u.Key.PrivateKey.D.Bytes(), nil, nil)
	key := make([]byte, 32)
	_, err := io.ReadFull(hkdfDerived, key)
	return key, err
}

func loadStoredKey(addr common.Address, filename, auth string) (*ethKs.Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := ethKs.DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}
