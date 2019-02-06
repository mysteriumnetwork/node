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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/requests"
)

// NewELKTransport creates transport allowing to send events to ELK through HTTP
func NewELKTransport(elkURL string, timeout time.Duration) Transport {
	return &elkTransport{http: newMysteriumHTTPTransport(timeout), elkURL: elkURL}
}

type elkTransport struct {
	http   mysterium.HTTPTransport
	elkURL string
}

func (transport *elkTransport) sendEvent(event event) error {
	req, err := requests.NewPostRequest(transport.elkURL, "/", event)
	if err != nil {
		return err
	}

	response, err := transport.http.Do(req)
	if err != nil {
		return err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error while reading elk response body: %v", err)
	}
	body := string(bodyBytes)

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected elk response status: %v, body: %v", response.Status, body)
	}

	if strings.ToUpper(body) != "OK" {
		return fmt.Errorf("unexpected response body: %v", body)
	}

	return nil
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
