package cli

import (
	"fmt"
	"bytes"
	"net/url"
	"net/http"
	"encoding/json"
	log "github.com/cihub/seelog"
	"errors"
)

type HttpClientInterface interface {
	Get(path string, values url.Values) (*http.Response, error)
	Post(path string, payload interface{}) (*http.Response, error)
	Put(path string, payload interface{}) (*http.Response, error)
	Delete(path string, payload interface{}) (*http.Response, error)
}

type HttpRequestInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewHttpClient(baseUrl string, logPrefix string, ua string) *HttpClient {
	httpClient := http.Client{
		Transport: &http.Transport{},
	}
	return &HttpClient{
		http:      &httpClient,
		baseUrl:   baseUrl,
		logPrefix: logPrefix,
		ua:        ua,
	}
}

type HttpClient struct {
	http      HttpRequestInterface
	baseUrl   string
	logPrefix string
	ua        string
}

func (client *HttpClient) Get(path string, values url.Values) (*http.Response, error) {
	fullPath := fmt.Sprintf("%v/%v?%v", client.baseUrl, path, values.Encode())
	return client.executeRequest("GET", fullPath, nil)
}

func (client *HttpClient) Post(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *HttpClient) Put(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("PUT", path, payload)
}

func (client *HttpClient) Delete(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("DELETE", path, payload)
}

func (client HttpClient) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	return client.executeRequest(method, client.baseUrl+"/"+path, payloadJson)
}

func (client *HttpClient) executeRequest(method, fullPath string, payloadJson []byte) (*http.Response, error) {
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
