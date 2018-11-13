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

package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/server/dto"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const (
	mysteriumAPILogPrefix = "[Mysterium.api] "
)

// HTTPTransport interface with single method do is extracted from net/transport.Client structure
type HTTPTransport interface {
	Do(*http.Request) (*http.Response, error)
}

func newHTTPTransport(requestTimeout time.Duration) HTTPTransport {
	return &http.Client{
		Transport: &http.Transport{
			//Don't reuse tcp connections for request - see ip/rest_resolver.go for details
			DisableKeepAlives: true,
		},
		Timeout: requestTimeout,
	}
}

type mysteriumAPI struct {
	http                HTTPTransport
	discoveryAPIAddress string
}

// NewClient creates Mysterium centralized api instance with real communication
func NewClient(discoveryAPIAddress string) Client {
	return &mysteriumAPI{
		newHTTPTransport(1 * time.Minute),
		discoveryAPIAddress,
	}
}

// RegisterIdentity registers given identity to discovery service
func (mApi *mysteriumAPI) RegisterIdentity(id identity.Identity, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "identities", dto.CreateIdentityRequest{
		Identity: id.Address,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Identity registered: ", id.Address)
	}
	return err
}

// RegisterProposal registers service proposal to discovery service
func (mApi *mysteriumAPI) RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "register_proposal", dto.NodeRegisterRequest{
		ServiceProposal: proposal,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal registered for node: ", proposal.ProviderID)
	}

	return err
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (mApi *mysteriumAPI) UnregisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "unregister_proposal", dto.ProposalUnregisterRequest{
		ProviderID:  proposal.ProviderID,
		ServiceType: proposal.ServiceType,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)

	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal unregistered for node: ", proposal.ProviderID)
	}

	return err
}

// PingProposal pings service proposal as being alive
func (mApi *mysteriumAPI) PingProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, "ping_proposal", dto.NodeStatsRequest{
		NodeKey:     proposal.ProviderID,
		ServiceType: proposal.ServiceType,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Proposal pinged for node: ", proposal.ProviderID)
	}
	return err
}

// FindProposals fetches currently active service proposals from discovery
func (mApi *mysteriumAPI) FindProposals(providerID string, serviceType string) ([]dto_discovery.ServiceProposal, error) {
	values := url.Values{}
	if providerID != "" {
		values.Set("node_key", providerID)
		values.Set("service_type", serviceType)
	}
	req, err := requests.NewGetRequest(mApi.discoveryAPIAddress, "proposals", values)
	if err != nil {
		return nil, err
	}

	var proposalsResponse dto.ProposalsResponse
	err = mApi.doRequestAndParseResponse(req, &proposalsResponse)
	if err != nil {
		return nil, err
	}

	log.Info(mysteriumAPILogPrefix, "Proposals fetched: ", len(proposalsResponse.Proposals))
	return proposalsResponse.Proposals, nil
}

// SendSessionStats sends session statistics
func (mApi *mysteriumAPI) SendSessionStats(sessionID session.ID, sessionStats dto.SessionStats, signer identity.Signer) error {
	path := fmt.Sprintf("sessions/%s/stats", sessionID)
	req, err := requests.NewSignedPostRequest(mApi.discoveryAPIAddress, path, sessionStats, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionID)
	}

	return nil
}

func (mApi *mysteriumAPI) doRequest(req *http.Request) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		log.Error(mysteriumAPILogPrefix, err)
		return err
	}
	defer resp.Body.Close()

	return ParseResponseError(resp)
}

func (mApi *mysteriumAPI) doRequestAndParseResponse(req *http.Request, responseValue interface{}) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		log.Error(mysteriumAPILogPrefix, err)
		return err
	}
	defer resp.Body.Close()

	err = ParseResponseError(resp)
	if err != nil {
		log.Error(mysteriumAPILogPrefix, err)
		return err
	}

	return ParseResponseJSON(resp, responseValue)
}
