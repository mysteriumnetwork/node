package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/MysteriumNetwork/node/server/dto"
)

const MYSTERIUM_API_URL = "https://mvp.mysterium.network:5000/v1"
const MYSTERIUM_API_CLIENT = "goclient-v0.1"
const MYSTERIUM_API_LOG_PREFIX = "[Mysterium.api] "

func NewClient() Client {
	httpClient := http.Client{
		Transport: &http.Transport{},
	}
	return &clientRest{
		httpClient: httpClient,
	}
}

type clientRest struct {
	httpClient http.Client
}

func (client *clientRest) NodeRegister(nodeKey, connectionConfig string) (err error) {
	response, err := client.doRequest("POST", "node_register", dto.NodeRegisterRequest{
		NodeKey:          nodeKey,
		ConnectionConfig: connectionConfig,
	})
	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Node registered: ", nodeKey)
	}

	return
}

func (client *clientRest) NodeSendStats(nodeKey string, sessionList []dto.SessionStats) (err error) {
	response, err := client.doRequest("POST", "node_send_stats", dto.NodeStatsRequest{
		NodeKey:  nodeKey,
		Sessions: sessionList,
	})
	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Node stats sent: ", nodeKey)
	}

	return nil
}

func (client *clientRest) SessionCreate(nodeKey string) (session dto.Session, err error) {
	response, err := client.doRequest("POST", "client_create_session", dto.SessionStartRequest{
		NodeKey: nodeKey,
	})
	if err == nil {
		defer response.Body.Close()
		err = parseResponseJson(response, &session)
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Session created: ", session.Id)
	}

	return
}

func (client *clientRest) SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error) {
	response, err := client.doRequest("POST", "client_send_stats", sessionStats)
	if err == nil {
		defer response.Body.Close()
		log.Info(MYSTERIUM_API_LOG_PREFIX, "Session stats sent: ", sessionId)
	}

	return nil
}

func (client *clientRest) doRequest(method string, path string, payload interface{}) (*http.Response, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Critical(MYSTERIUM_API_LOG_PREFIX, err)
		return nil, err
	}

	request, err := http.NewRequest(method, MYSTERIUM_API_URL+"/"+path, bytes.NewBuffer(payloadJson))
	request.Header.Set("User-Agent", MYSTERIUM_API_CLIENT)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(MYSTERIUM_API_LOG_PREFIX, err)
		return nil, err
	}

	response, err := client.httpClient.Do(request)
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
		return errors.New(fmt.Sprintf("Server response invalid: %s", response.Status))
	}

	return nil
}
