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

func NewClient() Client {
	return &mysteriumApi{
		&http.Client{
			Transport: &http.Transport{},
		},
	}
}

func (mApi *mysteriumApi) RegisterIdentity(identity identity.Identity) error {
	req, err := newPostRequest("identities", dto.CreateIdentityRequest{
		Identity: identity.Address,
	})
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Identity registered: ", identity)
	}
	return err
}

func (mApi *mysteriumApi) NodeRegister(proposal dto_discovery.ServiceProposal) error {
	req, err := newPostRequest("node_register", dto.NodeRegisterRequest{
		ServiceProposal: proposal,
	})
	if err != nil {
		return err
	}

	err = mApi.doRequest(req)
	if err == nil {
		log.Info(mysteriumApiLogPrefix, "Node registered: ", proposal.ProviderId)
	}

	return err
}

func (mApi *mysteriumApi) NodeSendStats(nodeKey string) error {
	req, err := newPostRequest("node_send_stats", dto.NodeStatsRequest{
		NodeKey: nodeKey,
		// TODO Refactor Node statistics with new `SessionStats` DTO
		Sessions: []dto.SessionStats{},
	})
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
	if err == nil {
		resp.Body.Close()
	}
	return parseResponseError(resp)
}

func (mApi *mysteriumApi) doRequestAndParseResponse(req *http.Request, responseValue interface{}) error {
	resp, err := mApi.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = parseResponseError(resp)
	if err != nil {
		return err
	}

	return parseResponseJson(resp, responseValue)
}
