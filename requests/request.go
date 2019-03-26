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
	"net/http"
	"net/url"

	"github.com/mysteriumnetwork/node/identity"
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

// NewSignedRequest signs payload and generates http request
func NewSignedRequest(httpMethod, apiURI, path string, requestBody interface{}, signer identity.Signer) (*http.Request, error) {
	var bodyBytes []byte = nil
	if requestBody != nil {
		var err error
		bodyBytes, err = encodeToJSON(requestBody)
		if err != nil {
			return nil, err
		}
	}

	signature, err := getBodySignature(bodyBytes, signer)
	if err != nil {
		return nil, err
	}

	req, err := newRequest(httpMethod, apiURI, path, bodyBytes)
	if err != nil {
		return nil, err
	}

	req.Header.Add(authenticationHeaderName, authenticationSchemaName+" "+signature.Base64())

	return req, nil
}

// NewSignedGetRequest signs empty message and generates http Get request
func NewSignedGetRequest(apiURI, path string, signer identity.Signer) (*http.Request, error) {
	return NewSignedRequest(http.MethodGet, apiURI, path, nil, signer)
}

// NewSignedPostRequest signs payload and generates http Post request
func NewSignedPostRequest(apiURI, path string, requestBody interface{}, signer identity.Signer) (*http.Request, error) {
	return NewSignedRequest(http.MethodPost, apiURI, path, requestBody, signer)
}

// NewSignedPutRequest signs payload and generates http Put request
func NewSignedPutRequest(apiURI, path string, requestBody interface{}, signer identity.Signer) (*http.Request, error) {
	return NewSignedRequest(http.MethodPut, apiURI, path, requestBody, signer)
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

func getBodySignature(bodyBytes []byte, signer identity.Signer) (identity.Signature, error) {
	var message = bodyBytes
	if message == nil {
		message = []byte("")
	}

	return signer.Sign(message)
}
