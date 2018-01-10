package server

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

var mysteriumApiUrl string

func newGetRequest(path string, params url.Values) (*http.Request, error) {
	pathWithQuery := fmt.Sprintf("%v?%v", path, params.Encode())
	return newRequest(http.MethodGet, pathWithQuery, nil)
}

func newPostRequest(path string, requestBody interface{}) (*http.Request, error) {
	bodyBytes, err := encodeToJson(requestBody)
	if err != nil {
		return nil, err
	}
	return newRequest(http.MethodPost, path, bodyBytes)
}

func newSignedPostRequest(path string, requestBody interface{}, signer identity.Signer) (*http.Request, error) {
	bodyBytes, err := encodeToJson(requestBody)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(bodyBytes)
	if err != nil {
		return nil, err
	}

	req, err := newRequest(http.MethodPost, path, bodyBytes)
	if err != nil {
		return nil, err
	}

	req.Header.Add(authenticationHeaderName, authenticationSchemaName+" "+signature.Base64())

	return req, nil
}

func encodeToJson(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func newRequest(method, path string, body []byte) (*http.Request, error) {

	fullUrl := fmt.Sprintf("%v/%v", mysteriumApiUrl, path)
	req, err := http.NewRequest(method, fullUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", mysteriumAgentName)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}
