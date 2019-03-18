/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package mysterium

import "github.com/mysteriumnetwork/node/market"

// CreateIdentityRequest represents JSON request for creating identity
type CreateIdentityRequest struct {
	Identity string `json:"identity"`
}

// UpdatePayoutInfoRequest represents JSON request for creating payout info request
type UpdatePayoutInfoRequest struct {
	EthAddress string `json:"payout_eth_address"`
}

// NodeRegisterRequest represents JSON for node registration request
type NodeRegisterRequest struct {
	ServiceProposal market.ServiceProposal `json:"service_proposal"`
}

// NodeStatsRequest represents JSON request for the node session stats information
type NodeStatsRequest struct {
	NodeKey     string         `json:"node_key"`
	ServiceType string         `json:"service_type"`
	Sessions    []SessionStats `json:"sessions"`
}

// ProposalUnregisterRequest represents request JSON for unregister a single proposal
type ProposalUnregisterRequest struct {
	// Unique identifier of a provider
	ProviderID  string `json:"provider_id"`
	ServiceType string `json:"service_type"`
}

// ProposalsRequest represents JSON request for the proposals
type ProposalsRequest struct {
	NodeKey     string `json:"node_key"`
	ServiceType string `json:"service_type"`
}

// ProposalsResponse represents JSON response for the list of proposals
type ProposalsResponse struct {
	Proposals []market.ServiceProposal `json:"proposals"`
}

// SessionStats mapped to json structure
type SessionStats struct {
	BytesSent       uint64 `json:"bytes_sent"`
	BytesReceived   uint64 `json:"bytes_received"`
	ProviderID      string `json:"provider_id"`
	ConsumerCountry string `json:"consumer_country"`
	ServiceType     string `json:"service_type"`
}
