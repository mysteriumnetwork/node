package server

import (
	"fmt"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"net/url"
)

const (
	mysteriumApiLogPrefix = "[Mysterium.api] "
)

//HttpTransport interface with single method do is extracted from net/transport.Client structure
type HttpTransport interface {
	Do(*http.Request) (*http.Response, error)
}

type mysteriumApi struct {
	http HttpTransport
}

//NewClient creates mysterium centralized api instance with real communcation
func NewClient() Client {
	return &mysteriumApi{
		&http.Client{
			Transport: &http.Transport{},
		},
	}
}

func (mApi *mysteriumApi) RegisterIdentity(identity identity.Identity, signer identity.Signer) error {
	req, err := newSignedPostRequest("identities", dto.CreateIdentityRequest{
		Identity: identity.Address,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Identity registered: ", identity)
	}
	return err
}

func (mApi *mysteriumApi) RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	req, err := newSignedPostRequest("node_register", dto.NodeRegisterRequest{
		ServiceProposal: proposal,
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Node registered: ", proposal.ProviderId)
	}

	return err
}

func (mApi *mysteriumApi) NodeSendStats(nodeKey string, signer identity.Signer) error {
	req, err := newSignedPostRequest("node_send_stats", dto.NodeStatsRequest{
		NodeKey: nodeKey,
		// TODO Refactor Node statistics with new `SessionStats` DTO
		Sessions: []dto.SessionStats{},
	}, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Node stats sent: ", nodeKey)
	}
	return err
}

func (mApi *mysteriumApi) FindProposals(nodeKey string) ([]dto_discovery.ServiceProposal, error) {
	values := url.Values{}
	values.Set("node_key", nodeKey)
	req, err := newGetRequest("proposals", values)
	if err != nil {
		return nil, err
	}

	var proposalsResponse dto.ProposalsResponse
	err = mApi.doRequestAndParseResponse(req, &proposalsResponse)
	if err != nil {
		return nil, err
	}

	log.Info(mysteriumApiLogPrefix, "FindProposals fetched: ", proposalsResponse.Proposals)

	return proposalsResponse.Proposals, nil
}

func (mApi *mysteriumApi) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) error {
	path := fmt.Sprintf("sessions/%s/stats", sessionId)
	req, err := newSignedPostRequest(path, sessionStats, signer)
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Session stats sent: ", sessionId)
	}

	return nil
}

func (mApi *mysteriumApi) doRequest(req *http.Request) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		log.Error(mysteriumApiLogPrefix, err)
		return err
	}
	defer resp.Body.Close()
	return parseResponseError(resp)
}

func (mApi *mysteriumApi) doRequestAndParseResponse(req *http.Request, responseValue interface{}) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		log.Error(mysteriumApiLogPrefix, err)
		return err
	}
	defer resp.Body.Close()

	err = parseResponseError(resp)
	if err != nil {
		log.Error(mysteriumApiLogPrefix, err)
		return err
	}

	return parseResponseJson(resp, responseValue)
}

func (mApi *mysteriumApi) AuthenticateClient(username, password string) (bool, error) {
	// TODO: implement
	return true, nil
}
