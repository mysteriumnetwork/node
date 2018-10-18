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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/cihub/seelog"
)

type httpClientInterface interface {
	Get(path string, values url.Values) (*http.Response, error)
	Post(path string, payload interface{}) (*http.Response, error)
	Put(path string, payload interface{}) (*http.Response, error)
	Delete(path string, payload interface{}) (*http.Response, error)
}

type httpRequestInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func newHTTPClient(baseURL string, logPrefix string, ua string) *httpClient {
	return &httpClient{
		http: &http.Client{
			Transport: &http.Transport{},
			Timeout:   time.Second * 120,
		},
		baseURL:   baseURL,
		logPrefix: logPrefix,
		ua:        ua,
	}
}

type httpClient struct {
	http      httpRequestInterface
	baseURL   string
	logPrefix string
	ua        string
}

func (client *httpClient) Get(path string, values url.Values) (*http.Response, error) {
	basePath := fmt.Sprintf("%v/%v", client.baseURL, path)

	var fullPath string
	params := values.Encode()
	if params == "" {
		fullPath = basePath
	} else {
		fullPath = fmt.Sprintf("%v?%v", basePath, params)
	}
	return client.executeRequest("GET", fullPath, nil)
}

func (client *httpClient) Post(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *httpClient) Put(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("PUT", path, payload)
}

func (client *httpClient) Delete(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("DELETE", path, payload)
}

func (client httpClient) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	return client.executeRequest(method, client.baseURL+"/"+path, payloadJSON)
}

func (client *httpClient) executeRequest(method, fullPath string, payloadJSON []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fullPath, bytes.NewBuffer(payloadJSON))
	request.Header.Set("User-Agent", client.ua)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	response, err := client.http.Do(request)

	if err != nil {
		log.Error(client.logPrefix, err)
		return response, err
	}

	err = parseResponseError(response)
	if err != nil {
		log.Error(client.logPrefix, err)
		return response, err
	}

	return response, nil
}

type errorBody struct {
	Message string `json:"message"`
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		//sometimes we can get json message with single "message" field which represents error - try to get that
		var parsedBody errorBody
		var message string
		err := parseResponseJSON(response, &parsedBody)
		if err != nil {
			message = err.Error()
		} else {
			message = parsedBody.Message
		}
		return fmt.Errorf("server response invalid: %s (%s). Possible error: %s", response.Status, response.Request.URL, message)
	}

	return nil
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(responseJSON, dto)
	if err != nil {
		return err
	}

	return nil
}
