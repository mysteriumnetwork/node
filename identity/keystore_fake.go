/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

type keyStoreFake struct {
	AccountsMock []accounts.Account
	ErrorMock    error
	LastHash     []byte
}

func (keyStore *keyStoreFake) Accounts() []accounts.Account {
	return keyStore.AccountsMock
}

func (keyStore *keyStoreFake) NewAccount(_ string) (accounts.Account, error) {
	if keyStore.ErrorMock != nil {
		return accounts.Account{}, keyStore.ErrorMock
	}

	accountNew := accounts.Account{
		Address: common.HexToAddress("0x000000000000000000000000000000000000bEEF"),
	}
	keyStore.AccountsMock = append(keyStore.AccountsMock, accountNew)

	return accountNew, nil
}

func (keyStore *keyStoreFake) Unlock(a accounts.Account, passphrase string) error {
	if keyStore.ErrorMock != nil {
		return keyStore.ErrorMock
	}

	return nil
}

func (keyStore *keyStoreFake) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	if keyStore.ErrorMock != nil {
		return []byte{}, keyStore.ErrorMock
	}

	keyStore.LastHash = hash
	return []byte("signed"), nil
}

func (keyStore *keyStoreFake) Find(a accounts.Account) (accounts.Account, error) {
	if keyStore.ErrorMock != nil {
		return accounts.Account{}, keyStore.ErrorMock
	}

	for _, acc := range keyStore.AccountsMock {
		if acc.Address == a.Address {
			return acc, nil
		}
	}

	return a, errors.New("account not found")
}
