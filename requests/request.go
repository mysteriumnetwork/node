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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mysterium/node/identity"
	"net/http"
	"net/url"
)

const (
	mysteriumAgentName       = "goclient-v0.1"
	authenticationHeaderName = "Authorization"
	authenticationSchemaName = "Signature"
)

// NewGetRequest generates http Get request
func NewGetRequest(apiURI, path string, params url.Values) (*http.Request, error) {
	pathWithQuery := fmt.Sprintf("%v?%v", path, params.Encode())
	return newRequest(http.MethodGet, apiURI, pathWithQuery, nil)
}

// NewPostRequest generates http Post request
func NewPostRequest(apiURI, path string, requestBody interface{}) (*http.Request, error) {
	bodyBytes, err := encodeToJSON(requestBody)
	if err != nil {
		return nil, err
	}
	return newRequest(http.MethodPost, apiURI, path, bodyBytes)
}

// NewSignedPostRequest signs payload and generates http Post request
func NewSignedPostRequest(apiURI, path string, requestBody interface{}, signer identity.Signer) (*http.Request, error) {
	bodyBytes, err := encodeToJSON(requestBody)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(bodyBytes)
	if err != nil {
		return nil, err
	}

	req, err := newRequest(http.MethodPost, apiURI, path, bodyBytes)
	if err != nil {
		return nil, err
	}

	req.Header.Add(authenticationHeaderName, authenticationSchemaName+" "+signature.Base64())

	return req, nil
}

func encodeToJSON(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func newRequest(method, apiURI, path string, body []byte) (*http.Request, error) {

	fullUrl := fmt.Sprintf("%v/%v", apiURI, path)
	req, err := http.NewRequest(method, fullUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", mysteriumAgentName)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}
