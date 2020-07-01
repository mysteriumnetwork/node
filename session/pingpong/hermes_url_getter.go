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

package pingpong

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// HermesURLGetter allows for fetching and storing of hermes urls.
type HermesURLGetter struct {
	Registry             common.Address
	bc                   bc
	loadedAddresses      map[common.Address]string
	loaddedAddressesLock sync.Mutex
}



// NewHermesURLGetter creates a new instance of hermes url getter.
func NewHermesURLGetter(
	bc bc,
	registry common.Address,
) *HermesURLGetter {
	return &HermesURLGetter{
		loadedAddresses: make(map[common.Address]string),
		Registry:        registry,
		bc:              bc,
	}
}

type bc interface {
	GetHermesURL(registryID, hermesID common.Address) (string, error)
}

const suffix = "api/v2"

func (hug *HermesURLGetter) normalizeAddress(address string) (string, error) {
	u, err := url.ParseRequestURI(address)
	if err != nil {
		return "", fmt.Errorf("Could not parse hermes URL: %w", err)
	}
	return fmt.Sprintf("%v://%v/%v", u.Scheme, u.Host, suffix), nil
}

// GetHermesURL fetches the hermes url from blockchain or local cache if it has already been loaded.
func (hug *HermesURLGetter) GetHermesURL(address common.Address) (string, error) {
	hug.loaddedAddressesLock.Lock()
	defer hug.loaddedAddressesLock.Unlock()

	add, ok := hug.loadedAddresses[address]
	if ok {
		return add, nil
	}

	add, err := hug.bc.GetHermesURL(hug.Registry, address)
	if err != nil {
		return "", err
	}

	add, err = hug.normalizeAddress(add)
	if err != nil {
		return "", err
	}

	hug.loadedAddresses[address] = add
	return add, nil
}
