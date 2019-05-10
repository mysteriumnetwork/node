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

import (
	"fmt"
	"net/url"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/session"
)

const (
	mysteriumAPILogPrefix = "[Mysterium.api] "
)

// MysteriumAPI provides access to mysterium owned central discovery service
type MysteriumAPI struct {
	http                requests.HTTPTransport
	discoveryAPIAddress string
}

// NewClient creates Mysterium centralized api instance with real communication
func NewClient(discoveryAPIAddress string) *MysteriumAPI {
	return &MysteriumAPI{
		requests.NewHTTPClient(1 * time.Minute),
		discoveryAPIAddress,
	}
}

// RegisterIdentity registers given identity to discovery service
func (mApi *MysteriumAPI) RegisterIdentity(id identity.Identity, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "identities", CreateIdentityRequest{
		Identity: id.Address,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Identity registered: ", id.Address)
	}
	return err
}

// GetPayoutInfo returns payout info from discovery service for identity
func (mApi *MysteriumAPI) GetPayoutInfo(id identity.Identity, signer identity.Signer) (*PayoutInfoResponse, error) {
	path := fmt.Sprintf("identities/%s/payout", id.Address)
	req, err := requests.NewSignedGetRequest(mApi.discoveryAPIAddress, path, signer)
	if err != nil {
		return nil, err
	}

	var payoutInfoResponse PayoutInfoResponse
	err = mApi.http.DoRequestAndParseResponse(req, &payoutInfoResponse)
	if err != nil {
		return nil, err
	}

	return &payoutInfoResponse, nil
}

// UpdatePayoutInfo registers given payout info next to identity to discovery service
func (mApi *MysteriumAPI) UpdatePayoutInfo(id identity.Identity, ethAddress string, signer identity.Signer) error {
	path := fmt.Sprintf("identities/%s/payout", id.Address)
	requestBody := UpdatePayoutInfoRequest{
		EthAddress: ethAddress,
	}
	req, err := requests.NewSignedPutRequest(mApi.discoveryAPIAddress, path, requestBody, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Payout address ", ethAddress, " registered")
	}
	return err
}

// RegisterProposal registers service proposal to discovery service
func (mApi *MysteriumAPI) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "register_proposal", NodeRegisterRequest{
		ServiceProposal: proposal,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal registered for node: ", proposal.ProviderID, " service type: ", proposal.ServiceType)
	}

	return err
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (mApi *MysteriumAPI) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "unregister_proposal", ProposalUnregisterRequest{
		ProviderID:  proposal.ProviderID,
		ServiceType: proposal.ServiceType,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)

	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal unregistered for node: ", proposal.ProviderID)
	}

	return err
}

// PingProposal pings service proposal as being alive
func (mApi *MysteriumAPI) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "ping_proposal", NodeStatsRequest{
		NodeKey:     proposal.ProviderID,
		ServiceType: proposal.ServiceType,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal pinged for node: ", proposal.ProviderID, " service type: ", proposal.ServiceType)
	}
	return err
}

// GetProposal fetches service proposal from discovery by exact ID
func (mApi *MysteriumAPI) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	proposals, err := mApi.doProposalRequest(url.Values{
		"node_key":     {id.ProviderID},
		"service_type": {id.ServiceType},
	})
	if err != nil {
		return nil, err
	}
	if len(proposals) == 0 {
		return nil, nil
	}

	return &proposals[0], nil
}

// FindProposals fetches currently active service proposals from discovery
func (mApi *MysteriumAPI) FindProposals(filter market.ProposalFilter) ([]market.ServiceProposal, error) {
	query := url.Values{}
	if filter.ProviderID != "" {
		query.Set("node_key", filter.ProviderID)
	}
	if filter.ServiceType != "" {
		query.Set("service_type", filter.ServiceType)
	}
	if filter.AccessPolicyID != "" {
		query.Set("access_policy[id]", filter.AccessPolicyID)
	}
	if filter.AccessPolicySource != "" {
		query.Set("access_policy[source]", filter.AccessPolicySource)
	}

	return mApi.doProposalRequest(query)
}

func (mApi *MysteriumAPI) doProposalRequest(query url.Values) ([]market.ServiceProposal, error) {
	req, err := requests.NewGetRequest(mApi.discoveryAPIAddress, "proposals", query)
	if err != nil {
		return nil, err
	}

	var proposalsResponse ProposalsResponse
	err = mApi.http.DoRequestAndParseResponse(req, &proposalsResponse)
	if err != nil {
		return nil, err
	}
	total := len(proposalsResponse.Proposals)
	supported := supportedProposalsOnly(proposalsResponse.Proposals)
	log.Trace(mysteriumAPILogPrefix, "Total proposals: ", total, " supported: ", len(supported))
	return supported, nil
}

// SendSessionStats sends session statistics
func (mApi *MysteriumAPI) SendSessionStats(sessionID session.ID, sessionStats SessionStats, signer identity.Signer) error {
	path := fmt.Sprintf("sessions/%s/stats", sessionID)
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, path, sessionStats, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionID)
	}

	return nil
}

func supportedProposalsOnly(proposals []market.ServiceProposal) (supported []market.ServiceProposal) {
	for _, proposal := range proposals {
		if proposal.IsSupported() {
			supported = append(supported, proposal)
		}
	}
	return
}
