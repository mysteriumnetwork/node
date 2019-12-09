/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"sync"
)

type accountantAddressProvider func(accountant common.Address) (common.Address, error)

// AccountantOwnerAddressCache gets and caches accountant addresses
type AccountantOwnerAddressCache struct {
	accountantOwners          map[common.Address]common.Address
	lock                      sync.Mutex
	accountantAddressProvider accountantAddressProvider
}

// NewAccountantOwnerAddressCache returns a new instance of accountant owner address cache
func NewAccountantOwnerAddressCache(accountantAddressProvider accountantAddressProvider) *AccountantOwnerAddressCache {
	return &AccountantOwnerAddressCache{
		accountantOwners:          make(map[common.Address]common.Address),
		accountantAddressProvider: accountantAddressProvider,
	}
}

// GetAccountantOwnerAddress gets the accountant address and keeps it in cache
func (aoac *AccountantOwnerAddressCache) GetAccountantOwnerAddress(id common.Address) (common.Address, error) {
	aoac.lock.Lock()
	defer aoac.lock.Unlock()
	if v, ok := aoac.accountantOwners[id]; ok {
		return v, nil
	}

	addr, err := aoac.accountantAddressProvider(id)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not get accountant address")
	}

	aoac.accountantOwners[id] = addr
	return addr, nil
}
