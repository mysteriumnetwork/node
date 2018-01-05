package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"net/url"
)

var mysteriumApiUrl string

const MYSTERIUM_API_CLIENT = "goclient-v0.1"
const MYSTERIUM_API_LOG_PREFIX = "[Mysterium.api] "

//HttpDoer is extracted one method interface from http.Client
type HttpDoer interface {
	Do(r *http.Request) (*http.Response, error)
}

type mysteriumRestClient struct {
	http HttpDoer
}

func NewClient() Client {
	httpClient := http.Client{
		Transport: &http.Transport{},
	}
	return &mysteriumRestClient{
		http: &httpClient,
	}
}

func (client *mysteriumRestClient) RegisterIdentity(identity identity.Identity) (err error) {
	response, err := client.doPostRequest("identities", dto.CreateIdentityRequest{
		Identity: identity.Address,
	})

	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Identity registered: ", identity)
	}

	return
}

func (client *mysteriumRestClient) NodeRegister(proposal dto_discovery.ServiceProposal) (err error) {
	response, err := client.doPostRequest("node_register", dto.NodeRegisterRequest{
		ServiceProposal: proposal,
	})

	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Node registered: ", proposal.ProviderId)
	}

	return
}

func (client *mysteriumRestClient) NodeSendStats(nodeKey string) (err error) {
	response, err := client.doPostRequest("node_send_stats", dto.NodeStatsRequest{
		NodeKey: nodeKey,
		// TODO Refactor Node statistics with new `SessionStats` DTO
		Sessions: []dto.SessionStats{},
	})
	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Node stats sent: ", nodeKey)
	}

	return nil
}

func (client *mysteriumRestClient) FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error) {
	values := url.Values{}
	values.Set("node_key", nodeKey)
	response, err := client.doGetRequest("proposals", values)

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

	log.Info(MYSTERIUM_API_LOG_PREFIX, "FindProposals fetched: ", proposals)

	return
}

func (client *mysteriumRestClient) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error) {
	path := fmt.Sprintf("sessions/%s/stats", sessionId)
	response, err := client.doPostRequest(path, sessionStats)
	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Session stats sent: ", sessionId)
	}

	return nil
}

func (client *mysteriumRestClient) doGetRequest(path string, values url.Values) (*http.Response, error) {
	fullPath := fmt.Sprintf("%v/%v?%v", mysteriumApiUrl, path, values.Encode())
	return client.executeRequest("GET", fullPath, nil)
}

func (client *mysteriumRestClient) doPostRequest(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *mysteriumRestClient) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Critical(MYSTERIUM_API_LOG_PREFIX, err)
		return nil, err
	}

	return client.executeRequest(method, mysteriumApiUrl+"/"+path, payloadJson)
}

func (client *mysteriumRestClient) executeRequest(method, fullPath string, payloadJson []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fullPath, bytes.NewBuffer(payloadJson))
	request.Header.Set("User-Agent", MYSTERIUM_API_CLIENT)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(MYSTERIUM_API_LOG_PREFIX, err)
		return nil, err
	}

	response, err := client.http.Do(request)

	if err != nil {
		log.Error(MYSTERIUM_API_LOG_PREFIX, err)
		return response, err
	}

	err = parseResponseError(response)
	if err != nil {
		log.Error(MYSTERIUM_API_LOG_PREFIX, err)
		return response, err
	}

	return response, nil
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

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("Server response invalid: %s (%s)", response.Status, response.Request.URL))
	}

	return nil
}
