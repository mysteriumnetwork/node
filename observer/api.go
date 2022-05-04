/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package observer

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/requests"
)

type httpClient interface {
	DoRequestAndParseResponse(req *http.Request, resp interface{}) error
}

// API is object which exposes observer API.
type API struct {
	req            httpClient
	url            string
	cachedResponse HermesesDataCache
	lock           sync.Mutex
}

const (
	hermesesEndpoint = "api/v1/observed/hermes"
)

// NewAPI returns a new API instance.
func NewAPI(hc httpClient, url string) *API {
	return &API{
		req: hc,
		url: strings.TrimSuffix(url, "/"),
	}
}

// HermesesResponse is returned from the observer hermeses endpoints and maps a slice of hermeses to its chain id.
type HermesesResponse map[int64][]HermesResponse

// HermesResponse describes a hermes.
type HermesResponse struct {
	HermesAddress      common.Address `json:"hermes_address"`
	Operator           common.Address `json:"operator"`
	Version            int            `json:"version"`
	Approved           bool           `json:"approved"`
	ChannelImplAddress common.Address `json:"channel_impl"`
	Fee                int            `json:"fee"`
	URL                string         `json:"url"`
}

// HermesesData is a map by chain id and hermes address to used to index hermes data.
type HermesesData map[int64]map[common.Address]HermesData

// HermesData describes the data belonging to a hermes.
type HermesData struct {
	Operator           common.Address
	Version            int
	Approved           bool
	ChannelImplAddress common.Address
	Fee                int
	URL                string
}

// HermesesDataCache describes a hermeses data cache.
type HermesesDataCache struct {
	HermesesData
	ValidUntil time.Time
}

// GetHermesesData returns a map by chain id and hermes address of all known hermeses.
func (a *API) GetHermesesData() (HermesesData, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if time.Now().Before(a.cachedResponse.ValidUntil) {
		return a.cachedResponse.HermesesData, nil
	}
	if a.url == "" {
		return nil, fmt.Errorf("no observer url set")
	}
	req, err := requests.NewGetRequest(a.url, hermesesEndpoint, nil)
	if err != nil {
		return nil, err
	}
	var resp HermesesResponse
	err = a.req.DoRequestAndParseResponse(req, &resp)
	if err != nil {
		return nil, err
	}
	hermesesData := make(map[int64]map[common.Address]HermesData)
	for chainId, hermeses := range resp {
		hermesesMap := make(map[common.Address]HermesData)
		for _, hermes := range hermeses {
			hermesesMap[hermes.HermesAddress] = HermesData{
				Operator:           hermes.Operator,
				Version:            hermes.Version,
				Approved:           hermes.Approved,
				ChannelImplAddress: hermes.ChannelImplAddress,
				Fee:                hermes.Fee,
				URL:                hermes.URL,
			}
		}
		hermesesData[chainId] = hermesesMap
	}
	a.cachedResponse = HermesesDataCache{
		HermesesData: hermesesData,
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	return hermesesData, err
}

// GetApprovedHermesAdresses returns a map by chain of all approved hermes addresses.
func (a *API) GetApprovedHermesAdresses() (map[int64][]common.Address, error) {
	resp, err := a.GetHermesesData()
	if err != nil {
		return nil, err
	}
	res := make(map[int64][]common.Address)
	for chainId, hermesesResp := range resp {
		hermeses := make([]common.Address, 0)
		for hermesAddress, hermesData := range hermesesResp {
			if hermesData.Approved {
				hermeses = append(hermeses, hermesAddress)
			}
		}
		res[chainId] = hermeses
	}
	return res, nil
}

// GetHermesData returns the data of a given hermes and chain id.
func (a *API) GetHermesData(chainId int64, hermesAddress common.Address) (HermesData, error) {
	hermesesData, err := a.GetHermesesData()
	if err != nil {
		return HermesData{}, err
	}
	hermesesForChainId, ok := hermesesData[chainId]
	if !ok {
		return HermesData{}, fmt.Errorf("no hermeses for chain id %d", chainId)
	}
	hermesData, ok := hermesesForChainId[hermesAddress]
	if !ok {
		return HermesData{}, fmt.Errorf("no data for hermes %s", hermesAddress.Hex())
	}
	return hermesData, nil
}
