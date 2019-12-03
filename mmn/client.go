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

package mmn

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

// NewClient returns MMN API client
func NewClient(httpClient *requests.HTTPClient, mmnAddress string, signer identity.SignerFactory) *client {
	return &client{
		httpClient: httpClient,
		mmnAddress: mmnAddress,
		signer:     signer,
	}
}

type client struct {
	httpClient *requests.HTTPClient
	mmnAddress string
	signer     identity.SignerFactory
}

func (m *client) RegisterNode(information *NodeInformation) error {
	id := identity.FromAddress(information.Identity)
	req, err := requests.NewSignedPostRequest(m.mmnAddress, "node", information, m.signer(id))
	if err != nil {
		return err
	}

	return m.httpClient.DoRequest(req)
}
