package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

var mysteriumApiUrl string

type mysteriumApi struct {
	json JsonClient
}

func NewClient() Client {
	return &mysteriumApi{
		json: NewJsonClient(
			mysteriumApiUrl,
			&http.Client{
				Transport: &http.Transport{},
			}),
	}
}

func (mApi *mysteriumApi) RegisterIdentity(identity identity.Identity) (err error) {
	response, err := mApi.json.Post("identities", dto.CreateIdentityRequest{
		Identity: identity.Address,
	})

	if err == nil {
		defer response.Body.Close()
		log.Info(mysteriumApiLogPrefix, "Identity registered: ", identity)
	}

	return
}

func (mApi *mysteriumApi) NodeRegister(proposal dto_discovery.ServiceProposal) (err error) {
	response, err := mApi.json.Post("node_register", dto.NodeRegisterRequest{
		ServiceProposal: proposal,
	})

	if err == nil {
		defer response.Body.Close()
		log.Info(mysteriumApiLogPrefix, "Node registered: ", proposal.ProviderId)
	}

	return
}

func (mApi *mysteriumApi) NodeSendStats(nodeKey string) (err error) {
	response, err := mApi.json.Post("node_send_stats", dto.NodeStatsRequest{
		NodeKey: nodeKey,
		// TODO Refactor Node statistics with new `SessionStats` DTO
		Sessions: []dto.SessionStats{},
	})
	if err == nil {
		defer response.Body.Close()
		log.Info(mysteriumApiLogPrefix, "Node stats sent: ", nodeKey)
	}

	return nil
}

func (mApi *mysteriumApi) FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error) {
	values := url.Values{}
	values.Set("node_key", nodeKey)
	response, err := mApi.json.Get("proposals", values)

	if err != nil {
		return
	}

	defer response.Body.Close()

	var proposalsResponse dto.ProposalsResponse
	err = parseResponseJson(response, &proposalsResponse)
	if err != nil {
		return
	}
	proposals = proposalsResponse.Proposals

	log.Info(mysteriumApiLogPrefix, "FindProposals fetched: ", proposals)

	return
}

func (mApi *mysteriumApi) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error) {
	path := fmt.Sprintf("sessions/%s/stats", sessionId)
	response, err := mApi.json.SignedPost(path, sessionStats, signer)
	if err == nil {
		defer response.Body.Close()
		log.Info(mysteriumApiLogPrefix, "Session stats sent: ", sessionId)
	}

	return nil
}

func parseResponseJson(response *http.Response, dto interface{}) error {
	responseJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(responseJson, dto)
	if err != nil {
		return err
	}

	return nil
}
