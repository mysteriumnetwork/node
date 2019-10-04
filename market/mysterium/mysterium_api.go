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
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"

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

	latestProposalsMux  sync.Mutex
	latestProposalsEtag string
	latestProposals     []market.ServiceProposal
}

// NewClient creates Mysterium centralized api instance with real communication
func NewClient(srcIP, discoveryAPIAddress string) *MysteriumAPI {
	return &MysteriumAPI{
		http:                requests.NewHTTPClient(srcIP, 20*time.Second),
		discoveryAPIAddress: discoveryAPIAddress,
		latestProposals:     []market.ServiceProposal{},
	}
}

// IdentityExists checks if given identity is registered in discovery
func (mApi *MysteriumAPI) IdentityExists(id identity.Identity, signer identity.Signer) (bool, error) {
	req, err := requests.NewSignedGetRequest(mApi.discoveryAPIAddress, fmt.Sprintf("identities/%s", id.Address), signer)
	if err != nil {
		return false, err
	}
	res, err := mApi.http.Do(req)
	if err != nil {
		return false, err
	}
	return res.StatusCode == 200, nil
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

// UpdateReferralInfo registers given referral code next to identity to discovery service
func (mApi *MysteriumAPI) UpdateReferralInfo(id identity.Identity, referralCode string, signer identity.Signer) error {
	path := fmt.Sprintf("identities/%s/referral", id.Address)
	requestBody := UpdateReferralInfoRequest{
		ReferralCode: referralCode,
	}
	req, err := requests.NewSignedPutRequest(mApi.discoveryAPIAddress, path, requestBody, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Referral code submitted for ", id.Address)
	}
	return err
}

// UpdateEmail registers given email next to identity to discovery service
func (mApi *MysteriumAPI) UpdateEmail(id identity.Identity, email string, signer identity.Signer) error {
	path := fmt.Sprintf("identities/%s/email", id.Address)
	requestBody := UpdateEmailRequest{
		Email: email,
	}
	req, err := requests.NewSignedPutRequest(mApi.discoveryAPIAddress, path, requestBody, signer)
	if err != nil {
		return err
	}

	err = mApi.http.DoRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Email submitted for ", id.Address)
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

// Proposals fetches currently active service proposals from discovery
func (mApi *MysteriumAPI) Proposals() ([]market.ServiceProposal, error) {
	return mApi.QueryProposals(ProposalsQuery{
		ServiceType:     "all",
		AccessPolicyAll: true,
	})
}

// QueryProposals fetches currently active service proposals from discovery - by given query filter
func (mApi *MysteriumAPI) QueryProposals(query ProposalsQuery) ([]market.ServiceProposal, error) {
	values := url.Values{}
	if query.NodeKey != "" {
		values.Set("node_key", query.NodeKey)
	}
	if query.ServiceType != "" {
		values.Set("service_type", query.ServiceType)
	}
	if query.AccessPolicyAll {
		values.Set("access_policy", "*")
	}
	if query.AccessPolicyID != "" {
		values.Set("access_policy[id]", query.AccessPolicyID)
	}
	if query.AccessPolicySource != "" {
		values.Set("access_policy[source]", query.AccessPolicySource)
	}
	if query.NodeType != "" {
		values.Set("node_type", query.NodeType)
	}

	req, err := requests.NewGetRequest(mApi.discoveryAPIAddress, "proposals", values)
	if err != nil {
		return nil, err
	}

	mApi.latestProposalsMux.Lock()
	defer mApi.latestProposalsMux.Unlock()
	req.Header.Add("If-None-Match", mApi.latestProposalsEtag)

	res, err := mApi.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch proposals")
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotModified {
		return mApi.latestProposals, nil
	}

	if err := requests.ParseResponseError(res); err != nil {
		return nil, err
	}

	var proposalsResponse ProposalsResponse
	if err := requests.ParseResponseJSON(res, &proposalsResponse); err != nil {
		return nil, errors.Wrap(err, "cannot parse proposals response")
	}

	mApi.latestProposalsEtag = res.Header.Get("ETag")
	total := len(proposalsResponse.Proposals)
	supported := supportedProposalsOnly(proposalsResponse.Proposals)
	mApi.latestProposals = supported
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
