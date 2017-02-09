package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/server/dto"
)

const MYSTERIUM_API_URL = "http://api.mysterium.network:5000/v1"
const MYSTERIUM_API_CLIENT = "goclient-v0.1"
const MYSTERIUM_API_LOG_PREFIX = "[OpenVPN.api] "

func NewClient() *client {
	httpClient := http.Client{
		Transport: &http.Transport{},
	}
	return &client{
		httpClient: httpClient,
	}
}

type client struct {
	httpClient http.Client
}

func (client *client) SessionCreate(nodeKey string) (session dto.Session, err error) {
	response, err := client.doRequest("POST", "client_create_session", dto.SessionStartRequest{
		NodeKey: nodeKey,
	})
	if err == nil {
		defer response.Body.Close()
		err = parseResponseJson(response, &session)
	}

	log.Info(MYSTERIUM_API_LOG_PREFIX, "Created new session: ", session.Id)
	return
}

func (client *client) doRequest(method string, path string, payload interface{}) (*http.Response, error) {
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

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Error(MYSTERIUM_API_LOG_PREFIX, err)
		return response, parseResponseError(response)
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
	return errors.New(fmt.Sprintf("Server response invalid: %s", response.Status))
}
