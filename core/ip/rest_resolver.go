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
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/requests"
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
		url:  url,
		http: requests.NewHTTPClient(timeout),
	}
}

type ipResponse struct {
	IP string `json:"IP"`
}

type clientRest struct {
	url  string
	http requests.HTTPTransport
}

func (client *clientRest) GetPublicIP() (string, error) {
	var ipResponse ipResponse

	request, err := requests.NewGetRequest(client.url, "", nil)
	request.Header.Set("User-Agent", apiClient)
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(ipAPILogPrefix, err)
		return "", err
	}

	err = client.http.DoRequestAndParseResponse(request, &ipResponse)
	if err != nil {
		return "", err
	}

	log.Trace(ipAPILogPrefix, "IP detected: ", ipResponse.IP)
	return ipResponse.IP, nil
}

func (client *clientRest) GetOutboundIP() (string, error) {
	ip, err := GetOutbound()
	return ip.String(), err
}
