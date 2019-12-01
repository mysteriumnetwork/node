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
	"net"
	"net/http"
	"time"

	"github.com/mysteriumnetwork/node/logconfig/httptrace"
	"github.com/pkg/errors"
)

const (
	// DefaultTimeout is a default http client timeout.
	DefaultTimeout = 20 * time.Second
)

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(srcIP string, timeout time.Duration) *HTTPClient {
	ipAddress := net.ParseIP(srcIP)
	localIPAddress := net.TCPAddr{IP: ipAddress}

	return &HTTPClient{
		Client: &http.Client{
			Transport: newTransport(&localIPAddress),
			Timeout:   timeout,
		},
	}
}

// HTTPClient describes a client for performing HTTP requests.
type HTTPClient struct {
	*http.Client
}

// DoRequest performs HTTP requests and parses error without returning response.
func (c *HTTPClient) DoRequest(req *http.Request) error {
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return ParseResponseError(response)
}

// DoRequestAndParseResponse performs HTTP requests and response from JSON.
func (c *HTTPClient) DoRequestAndParseResponse(req *http.Request, resp interface{}) error {
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	httptrace.TraceRequestResponse(req, response)

	err = ParseResponseError(response)
	if err != nil {
		return err
	}

	return ParseResponseJSON(response, &resp)
}

// ParseResponseJSON parses http.Response into given struct.
func ParseResponseJSON(response *http.Response, dto interface{}) error {
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

// ParseResponseError parses http.Response error.
func ParseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.Errorf("server response invalid: %s (%s)", response.Status, response.Request.URL)
	}

	return nil
}
