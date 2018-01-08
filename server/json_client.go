package server

import (
	"bytes"
	"encoding/json"
	"errors"
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

var noSignature = identity.SignatureBytes(nil)

type HttpClient interface {
	DoGet(path string, values url.Values) (*http.Response, error)
	DoPost(path string, body interface{}) (*http.Response, error)
	DoSignedPost(path string, body interface{}, signer identity.Signer) (*http.Response, error)
}

//HttpTransport interface with single method do is extracted from net/transport.Client structure
type HttpTransport interface {
	Do(*http.Request) (*http.Response, error)
}

func NewJsonClient(baseUrl string, transport HttpTransport) *jsonHttpClient {
	return &jsonHttpClient{
		baseUrl,
		transport,
	}
}

type jsonHttpClient struct {
	baseApiUrl string
	transport  HttpTransport
}

func (jhc *jsonHttpClient) DoGet(path string, values url.Values) (*http.Response, error) {
	pathWithQuery := fmt.Sprintf("%v?%v", path, values.Encode())
	return jhc.executeRequest(http.MethodGet, pathWithQuery, nil, noSignature)

}

func (jhc *jsonHttpClient) DoPost(path string, body interface{}) (*http.Response, error) {
	return jhc.doSignedPayloadRequest(http.MethodPost, path, body, noOpSigner{})
}

func (jhc *jsonHttpClient) DoSignedPost(path string, body interface{}, signer identity.Signer) (*http.Response, error) {
	return jhc.doSignedPayloadRequest(http.MethodPost, path, body, signer)
}

func (jhc *jsonHttpClient) doSignedPayloadRequest(method string, path string, body interface{}, signer identity.Signer) (*http.Response, error) {
	payloadJson, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	signature, err := signer.Sign(payloadJson)
	if err != nil {
		return nil, err
	}
	return jhc.executeRequest(method, path, payloadJson, signature)
}

func (jhc *jsonHttpClient) executeRequest(method string, path string, body []byte, signature identity.Signature) (*http.Response, error) {
	fullPath := fmt.Sprintf("%v/%v", jhc.baseApiUrl, path)
	req, err := http.NewRequest(method, fullPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", mysteriumAgentName)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if !signature.EqualsTo(noSignature) {
		req.Header.Add(authenticationHeaderName, authenticationSchemaName+" "+signature.Base64())
	}

	resp, err := jhc.transport.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, parseResponseError(resp)
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("Server response invalid: %s (%s)", response.Status, response.Request.URL))
	}

	return nil
}

type noOpSigner struct {
}

func (_ noOpSigner) Sign(message []byte) (identity.Signature, error) {
	return noSignature, nil
}
