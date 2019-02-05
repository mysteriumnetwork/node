/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package events

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/requests"
)

// NewELKTransport creates transport allowing to send events to ELK through HTTP
func NewELKTransport(elkUrl string, timeout time.Duration) Transport {
	return &elkTransport{http: newMysteriumHTTPTransport(timeout), elkUrl: elkUrl}
}

type elkTransport struct {
	http   mysterium.HTTPTransport
	elkUrl string
}

func (transport *elkTransport) sendEvent(event event) error {
	fmt.Printf("Sending event %v\n", event)
	req, err := requests.NewPostRequest(transport.elkUrl, "/", event)
	if err != nil {
		return err
	}

	response, err := transport.http.Do(req)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New("Unexpected elk response status: " + response.Status)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	body := string(bodyBytes)
	if strings.ToUpper(body) != "OK" {
		return errors.New("Unexpected response body: " + body)
	}

	return err
}

func newMysteriumHTTPTransport(timeout time.Duration) mysterium.HTTPTransport {
	return &http.Client{
		Transport: &http.Transport{
			//Don't reuse tcp connections for request - see ip/rest_resolver.go for details
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	}
}

type event struct {
	Application applicationInfo `json:"application"`
	EventName   string          `json:"eventName"`
	CreatedAt   int64           `json:"createdAt"`
	Context     interface{}     `json:"context"`
}

type applicationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
