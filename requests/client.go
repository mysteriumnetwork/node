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

package requests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mysteriumnetwork/node/logconfig/httptrace"
	"github.com/pkg/errors"
)

// HTTPTransport describes a client for performing HTTP requests.
type HTTPTransport interface {
	Do(req *http.Request) (*http.Response, error)
	DoRequest(req *http.Request) error
	DoRequestAndParseResponse(req *http.Request, resp interface{}) error
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(timeout time.Duration) *client {
	return &client{
		&http.Client{Transport: &http.Transport{
			//dont cache tcp connections - first requests after state change (direct -> tunneled and vice versa) will always fail
			//as stale tcp states are not closed after switch. Probably some kind of CloseIdleConnections will help in the future
			DisableKeepAlives: true,
		},
			Timeout: timeout,
		},
	}
}

type client struct {
	*http.Client
}

func (c *client) DoRequest(req *http.Request) error {
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return parseResponseError(response)
}

func (c *client) DoRequestAndParseResponse(req *http.Request, resp interface{}) error {
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	httptrace.TraceRequestResponse(req, response)

	err = parseResponseError(response)
	if err != nil {
		return err
	}

	return parseResponseJSON(response, &resp)
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
		return errors.Errorf("server response invalid: %s (%s)", response.Status, response.Request.URL)
	}

	return nil
}
