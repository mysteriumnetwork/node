package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"net/http"
	"net/url"
)

type httpClientInterface interface {
	Get(path string, values url.Values) (*http.Response, error)
	Post(path string, payload interface{}) (*http.Response, error)
	Put(path string, payload interface{}) (*http.Response, error)
	Delete(path string, payload interface{}) (*http.Response, error)
}

type httpRequestInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func newHttpClient(baseUrl string, logPrefix string, ua string) *httpClient {
	return &httpClient{
		http: &http.Client{
			Transport: &http.Transport{},
		},
		baseUrl:   baseUrl,
		logPrefix: logPrefix,
		ua:        ua,
	}
}

type httpClient struct {
	http      httpRequestInterface
	baseUrl   string
	logPrefix string
	ua        string
}

func (client *httpClient) Get(path string, values url.Values) (*http.Response, error) {
	fullPath := fmt.Sprintf("%v/%v?%v", client.baseUrl, path, values.Encode())
	return client.executeRequest("GET", fullPath, nil)
}

func (client *httpClient) Post(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *httpClient) Put(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("PUT", path, payload)
}

func (client *httpClient) Delete(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("DELETE", path, payload)
}

func (client httpClient) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	return client.executeRequest(method, client.baseUrl+"/"+path, payloadJson)
}

func (client *httpClient) executeRequest(method, fullPath string, payloadJson []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fullPath, bytes.NewBuffer(payloadJson))
	request.Header.Set("User-Agent", client.ua)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	response, err := client.http.Do(request)

	if err != nil {
		log.Error(client.logPrefix, err)
		return response, err
	}

	err = parseResponseError(response)
	if err != nil {
		log.Error(client.logPrefix, err)
		return response, err
	}

	return response, nil
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("Server response invalid: %s (%s)", response.Status, response.Request.URL))
	}

	return nil
}
