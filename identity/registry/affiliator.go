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

package registry

import (
	"fmt"
	"math/big"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/requests"
)

// Affiliator allows for convenient calls to the affiliator service
type Affiliator struct {
	httpClient      *requests.HTTPClient
	endpointAddress string
}

// NewAffiliator creates and returns new Affiliator instance
func NewAffiliator(httpClient *requests.HTTPClient, endpointAddress string) *Affiliator {
	return &Affiliator{
		httpClient:      httpClient,
		endpointAddress: endpointAddress,
	}
}

// RegistrationTokenReward returns the amount of MYST rewarder for token used.
func (t *Affiliator) RegistrationTokenReward(token string) (*big.Int, error) {
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("register-reward/token/%s", token), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token reward amount")
	}

	var resp struct {
		Reward string `json:"reward"`
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token reward amount")
	}
	reward, _ := new(big.Int).SetString(resp.Reward, 10)

	return reward, err
}
