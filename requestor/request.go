package requestor

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
