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
	"github.com/mysteriumnetwork/payments/observer"
	"github.com/rs/zerolog/log"
)

// HermesURLGetter allows for fetching and storing of hermes urls.
type HermesURLGetter struct {
	addressProvider      addressProvider
	bc                   bc
	loadedAddresses      map[int64]map[common.Address]string
	loaddedAddressesLock sync.Mutex
	observer             observerApi
}

type addressProvider interface {
	GetActiveChannelAddress(chainID int64, id common.Address) (common.Address, error)
	GetArbitraryChannelAddress(hermes, registry, channel common.Address, id common.Address) (common.Address, error)
	GetActiveChannelImplementation(chainID int64) (common.Address, error)
	GetChannelImplementationForHermes(chainID int64, hermes common.Address) (common.Address, error)
	GetMystAddress(chainID int64) (common.Address, error)
	GetActiveHermes(chainID int64) (common.Address, error)
	GetRegistryAddress(chainID int64) (common.Address, error)
	GetKnownHermeses(chainID int64) ([]common.Address, error)
	GetHermesChannelAddress(chainID int64, id, hermesAddr common.Address) (common.Address, error)
}

type observerApi interface {
	GetHermeses(f *observer.HermesFilter) (observer.HermesesResponse, error)
	GetHermesData(chainId int64, hermesAddress common.Address) (*observer.HermesResponse, error)
}

// NewHermesURLGetter creates a new instance of hermes url getter.
func NewHermesURLGetter(
	bc bc,
	addressProvider addressProvider,
	observer observerApi,
) *HermesURLGetter {
	return &HermesURLGetter{
		loadedAddresses: make(map[int64]map[common.Address]string),
		addressProvider: addressProvider,
		bc:              bc,
		observer:        observer,
	}
}

type bc interface {
	GetHermesURL(chainID int64, registryID, hermesID common.Address) (string, error)
}

const suffix = "api/v2"

func (hug *HermesURLGetter) normalizeAddress(address string) (string, error) {
	u, err := url.ParseRequestURI(address)
	if err != nil {
		return "", fmt.Errorf("could not parse hermes URL: %w", err)
	}
	return fmt.Sprintf("%v://%v/%v", u.Scheme, u.Host, suffix), nil
}

// GetHermesURL fetches the hermes url from blockchain, observer or local cache if it has already been loaded.
func (hug *HermesURLGetter) GetHermesURL(chainID int64, address common.Address) (string, error) {
	hug.loaddedAddressesLock.Lock()
	defer hug.loaddedAddressesLock.Unlock()

	addresses, ok := hug.loadedAddresses[chainID]
	if ok {
		v, ok := addresses[address]
		if ok {
			return v, nil
		}
	} else {
		hug.loadedAddresses[chainID] = make(map[common.Address]string, 0)
	}

	url, err := hug.getHermesURLBC(chainID, address)
	if err != nil {
		log.Err(err).Fields(map[string]any{
			"chain_id":  chainID,
			"hermes_id": address.Hex(),
		}).Msg("failed to get hermes url from blockchain, using fallback")
		url, err = hug.getHermesURLObserver(chainID, address)
		if err != nil {
			return "", err
		}
	}
	url, err = hug.normalizeAddress(url)
	if err != nil {
		return "", err
	}
	hug.loadedAddresses[chainID][address] = url
	return url, nil
}

func (hug *HermesURLGetter) getHermesURLBC(chainID int64, address common.Address) (string, error) {
	registry, err := hug.addressProvider.GetRegistryAddress(chainID)
	if err != nil {
		return "", err
	}
	return hug.bc.GetHermesURL(chainID, registry, address)
}

func (hug *HermesURLGetter) getHermesURLObserver(chainID int64, address common.Address) (string, error) {
	hermesData, err := hug.observer.GetHermesData(chainID, address)
	if err != nil {
		return "", err
	}
	return hug.normalizeAddress(hermesData.URL)
}
