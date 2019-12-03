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

// NodeInformationDto contains node information to be sent to MMN
type NodeInformationDto struct {
	MACAddress  string `json:"mac_address"` // SHA256 hash
	LocalIP     string `json:"local_ip"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	NodeVersion string `json:"node_version"`
	Identity    string `json:"identity"`
	VendorID    string `json:"vendor_id"`
	IsProvider  bool   `json:"is_provider"`
	IsClient    bool   `json:"is_client"`
}

// NodeTypeDto contains node type information to be sent to MMN
type NodeTypeDto struct {
	IsProvider bool   `json:"is_provider"`
	IsClient   bool   `json:"is_client"`
	Identity   string `json:"identity"`
}

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

func (m *client) RegisterNode(info *NodeInformationDto) error {
	id := identity.FromAddress(info.Identity)
	req, err := requests.NewSignedPostRequest(m.mmnAddress, "node", info, m.signer(id))
	if err != nil {
		return err
	}

	return m.httpClient.DoRequest(req)
}

func (m *client) UpdateNodeType(info *NodeInformationDto) error {
	id := identity.FromAddress(info.Identity)
	nodeType := NodeTypeDto{
		IsProvider: info.IsProvider,
		IsClient:   info.IsClient,
		Identity:   info.Identity,
	}

	req, err := requests.NewSignedPostRequest(
		m.mmnAddress,
		"node/type",
		nodeType,
		m.signer(id),
	)
	if err != nil {
		return err
	}

	return m.httpClient.DoRequest(req)
}
