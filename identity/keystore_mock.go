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
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	ethKs "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type mockKeystore struct {
	keys map[common.Address]MockKey
	lock sync.Mutex
}

// MockKeys represents the mocked keys
var MockKeys = map[common.Address]MockKey{
	common.HexToAddress("53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"): {
		PkHex: "6f88637b68ee88816e73f663aef709d7009836c98ae91ef31e3dfac7be3a1657",
		Pass:  "",
	},
}

// NewMockKeystore returns empty mock keystore
func NewMockKeystore() *mockKeystore {
	return NewMockKeystoreWith(map[common.Address]MockKey{})
}

// NewMockKeystoreWith returns a new mock keystore with specified keys
func NewMockKeystoreWith(keys map[common.Address]MockKey) *mockKeystore {
	copied := make(map[common.Address]MockKey)
	for key, value := range keys {
		copied[key] = value
	}
	return &mockKeystore{
		keys: copied,
	}
}

// MockKey represents a mocked key
type MockKey struct {
	PkHex      string
	Pass       string
	pk         *ecdsa.PrivateKey
	isUnlocked bool
}

func (mk *mockKeystore) Accounts() []accounts.Account {
	mk.lock.Lock()
	defer mk.lock.Unlock()
	res := make([]accounts.Account, 0)
	for k := range mk.keys {
		res = append(res, accounts.Account{
			Address: k,
		})
	}
	return res
}

func (mk *mockKeystore) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	mk.lock.Lock()
	defer mk.lock.Unlock()

	if v, ok := mk.keys[a.Address]; ok {
		if !v.isUnlocked {
			return nil, ethKs.ErrLocked
		}
		return crypto.Sign(hash, v.pk)
	}
	return nil, ethKs.ErrNoMatch
}

func (mk *mockKeystore) Export(a accounts.Account, passphrase, newPassphrase string) (keyJSON []byte, err error) {
	mk.lock.Lock()
	defer mk.lock.Unlock()
	if v, ok := mk.keys[a.Address]; ok {
		if v.Pass != passphrase {
			return nil, ethKs.ErrDecrypt
		}
		return common.Hex2Bytes(v.PkHex), nil
	}
	return nil, ethKs.ErrNoMatch
}

func (mk *mockKeystore) NewAccount(passphrase string) (accounts.Account, error) {
	mk.lock.Lock()
	defer mk.lock.Unlock()

	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), crand.Reader)
	if err != nil {
		return accounts.Account{}, err
	}

	address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	mk.keys[address] = MockKey{
		Pass:       passphrase,
		PkHex:      hex.EncodeToString(crypto.FromECDSA(privateKeyECDSA)),
		pk:         privateKeyECDSA,
		isUnlocked: false,
	}
	return accounts.Account{
		Address: address,
	}, nil
}

func (mk *mockKeystore) Unlock(a accounts.Account, passphrase string) error {
	mk.lock.Lock()
	defer mk.lock.Unlock()

	if v, ok := mk.keys[a.Address]; ok {
		if v.isUnlocked {
			return nil
		}

		if v.Pass != passphrase {
			return ethKs.ErrDecrypt
		}

		pk, err := crypto.HexToECDSA(v.PkHex)
		if err != nil {
			return err
		}

		v.isUnlocked = true
		v.pk = pk
		mk.keys[a.Address] = v
		return nil
	}

	return ethKs.ErrNoMatch
}

func (mk *mockKeystore) Lock(addr common.Address) error {
	mk.lock.Lock()
	defer mk.lock.Unlock()

	if v, ok := mk.keys[addr]; ok {
		v.isUnlocked = false
		mk.keys[addr] = v
	}
	return nil
}

func (mk *mockKeystore) Find(a accounts.Account) (accounts.Account, error) {
	mk.lock.Lock()
	defer mk.lock.Unlock()

	if _, ok := mk.keys[a.Address]; ok {
		return a, nil
	}
	return accounts.Account{}, ethKs.ErrNoMatch
}

// MockDecryptFunc represents the mock decrypt func
var MockDecryptFunc = func(keyjson []byte, auth string) (*ethKs.Key, error) {
	pk, err := crypto.HexToECDSA(common.Bytes2Hex(keyjson))
	if err != nil {
		return nil, err
	}
	return &ethKs.Key{
		PrivateKey: pk,
	}, nil
}
