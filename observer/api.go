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
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/requests"
)

// API is object which exposes observer API.
type API struct {
	req *requests.HTTPClient
	url string
}

const (
	hermesesEndpoint = "api/v1/observed/hermes"
)

// NewAPI returns a new API instance.
func NewAPI(hc *requests.HTTPClient, url string) *API {
	return &API{
		req: hc,
		url: strings.TrimSuffix(url, "/"),
	}
}

// HermesesResponse is returned from the observer hermeses endpoints and maps a slice of hermeses to its chain id.
type HermesesResponse map[int64][]HermesResponse

// HermesResponse describes a hermes.
type HermesResponse struct {
	HermesAddress common.Address `json:"hermes_address"`
	Operator      common.Address `json:"operator"`
	Version       int            `json:"version"`
	Approved      bool           `json:"approved"`
}

// GetHermeses returns a map by chain of all known hermeses.
func (a *API) GetHermeses() (*HermesesResponse, error) {
	if a.url == "" {
		return nil, fmt.Errorf("no observer url set")
	}
	req, err := requests.NewGetRequest(a.url, hermesesEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp HermesesResponse
	return &resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// GetApprovedHermesAdresses returns a map by chain of all approved hermes addresses.
func (a *API) GetApprovedHermesAdresses() (map[int64][]common.Address, error) {
	resp, err := a.GetHermeses()
	if err != nil {
		return nil, err
	}
	res := make(map[int64][]common.Address)
	for chainId, hermesesResp := range *resp {
		hermeses := make([]common.Address, 0)
		for _, h := range hermesesResp {
			if h.Approved {
				hermeses = append(hermeses, h.HermesAddress)
			}
		}
		res[chainId] = hermeses
	}
	return res, nil
}
