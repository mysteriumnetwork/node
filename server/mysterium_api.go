package server

import (
	"fmt"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/requestor"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"net/url"
)

const (
	mysteriumAPILogPrefix = "[Mysterium.api] "
)

//HttpTransport interface with single method do is extracted from net/transport.Client structure
type HttpTransport interface {
	Do(*http.Request) (*http.Response, error)
}

type mysteriumAPI struct {
	http                HttpTransport
	discoveryAPIAddress string
}

// NewClient creates Mysterium centralized api instance with real communication
func NewClient(discoveryAPIAddress string) Client {
	return &mysteriumAPI{
		&http.Client{
			Transport: &http.Transport{},
		},
		discoveryAPIAddress,
	}
}

func (mApi *mysteriumAPI) RegisterIdentity(id identity.Identity, signer identity.Signer) error {
	req, err := requestor.NewSignedPostRequest(mApi.discoveryAPIAddress, "identities", dto.CreateIdentityRequest{
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

func (mApi *mysteriumAPI) RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := requestor.NewSignedPostRequest(mApi.discoveryAPIAddress, "register_proposal", dto.NodeRegisterRequest{
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

func (mApi *mysteriumAPI) PingProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := requestor.NewSignedPostRequest(mApi.discoveryAPIAddress, "ping_proposal", dto.NodeStatsRequest{
		NodeKey: proposal.ProviderID,
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

func (mApi *mysteriumAPI) FindProposals(providerID string) ([]dto_discovery.ServiceProposal, error) {
	values := url.Values{}
	if providerID != "" {
		values.Set("node_key", providerID)
	}

	req, err := requestor.NewGetRequest(mApi.discoveryAPIAddress, "proposals", values)
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

func (mApi *mysteriumAPI) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) error {
	path := fmt.Sprintf("sessions/%s/stats", sessionId)
	req, err := requestor.NewSignedPostRequest(mApi.discoveryAPIAddress, path, sessionStats, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionId)
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

	return parseResponseError(resp)
}

func (mApi *mysteriumAPI) doRequestAndParseResponse(req *http.Request, responseValue interface{}) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		log.Error(mysteriumAPILogPrefix, err)
		return err
	}
	defer resp.Body.Close()

	err = parseResponseError(resp)
	if err != nil {
		log.Error(mysteriumAPILogPrefix, err)
		return err
	}

	return parseResponseJson(resp, responseValue)
}
