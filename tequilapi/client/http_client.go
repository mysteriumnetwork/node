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
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/requests"
)

type httpClientInterface interface {
	SetToken(token string)
	Get(path string, values url.Values) (*http.Response, error)
	Post(path string, payload interface{}) (*http.Response, error)
	Put(path string, payload interface{}) (*http.Response, error)
	Delete(path string, payload interface{}) (*http.Response, error)
}

type httpRequestInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func newHTTPClient(baseURL string, ua string) *httpClient {
	return &httpClient{
		http:    requests.NewHTTPClient("0.0.0.0", 100*time.Second),
		baseURL: baseURL,
		ua:      ua,
	}
}

type httpClient struct {
	http      httpRequestInterface
	authToken string
	baseURL   string
	ua        string
}

func (client *httpClient) SetToken(token string) {
	client.authToken = token
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
		log.Error().Err(err).Msg("")
		return nil, err
	}

	return client.executeRequest(method, client.baseURL+"/"+path, payloadJSON)
}

func (client *httpClient) executeRequest(method, fullPath string, payloadJSON []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fullPath, bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Error().Err(err).Msg("")
		return nil, err
	}
	request.Header.Set("User-Agent", client.ua)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if client.authToken != "" {
		request.Header.Set("Authorization", "Bearer "+client.authToken)
	}

	response, err := client.http.Do(request)
	if err != nil {
		log.Error().Err(err).Msg("")
		return response, err
	}

	err = parseResponseError(response)
	if err != nil {
		log.Error().Err(err).Msg("")
		return response, err
	}

	return response, nil
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return apierror.Parse(response)
	}
	return nil
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	b := bytes.NewBuffer(make([]byte, 0))
	reader := io.TeeReader(response.Body, b)

	if err := json.NewDecoder(reader).Decode(dto); err != nil {
		return err
	}

	defer response.Body.Close()

	// NopCloser returns a ReadCloser with a no-op Close method wrapping the provided Reader r.
	// parseResponseError "empties" the contents of an errored response
	// this way the response can be read and parsed again further down the line
	response.Body = io.NopCloser(b)

	return nil
}
