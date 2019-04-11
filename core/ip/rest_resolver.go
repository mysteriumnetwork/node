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

package ip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
)

const apiClient = "goclient-v0.1"
const ipAPILogPrefix = "[ip-detector.api] "

// NewResolver creates new ip-detector resolver with default timeout of one minute
func NewResolver(url string) Resolver {
	return NewResolverWithTimeout(url, 1*time.Minute)
}

// NewResolverWithTimeout creates new ip-detector resolver with specified timeout
func NewResolverWithTimeout(url string, timeout time.Duration) Resolver {
	return &clientRest{
		url: url,
		httpClient: http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				//dont cache tcp connections - first requests after state change (direct -> tunneled and vice versa) will always fail
				//as stale tcp states are not closed after switch. Probably some kind of CloseIdleConnections will help in the future
				DisableKeepAlives: true,
			},
		},
	}
}

type ipResponse struct {
	IP string `json:"IP"`
}

type clientRest struct {
	url        string
	httpClient http.Client
}

func (client *clientRest) GetPublicIP() (string, error) {
	var ipResponse ipResponse

	request, err := http.NewRequest("GET", client.url, nil)
	request.Header.Set("User-Agent", apiClient)
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(ipAPILogPrefix, err)
		return "", err
	}

	err = client.doRequest(request, &ipResponse)
	if err != nil {
		return "", err
	}

	log.Info(ipAPILogPrefix, "IP detected: ", ipResponse.IP)
	return ipResponse.IP, nil
}

func (client *clientRest) GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Info("[Detect Outbound IP] ", "IP detected: ", localAddr.IP.String())
	return localAddr.IP.String(), nil
}

func (client *clientRest) doRequest(request *http.Request, responseDto interface{}) error {
	response, err := client.httpClient.Do(request)
	if err != nil {
		log.Error(ipAPILogPrefix, err)
		return err
	}
	defer response.Body.Close()

	err = parseResponseError(response)
	if err != nil {
		log.Error(ipAPILogPrefix, err)
		return err
	}

	return parseResponseJSON(response, &responseDto)
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

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("server response invalid: %s (%s)", response.Status, response.Request.URL)
	}

	return nil
}
