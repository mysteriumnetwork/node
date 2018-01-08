package http

import (
	"fmt"
	"bytes"
	"net/url"
	"net/http"
	"encoding/json"
	log "github.com/cihub/seelog"
	"errors"
)

func NewClient(baseUrl string, logPrefix string, ua string) *Client {
	httpClient := http.Client{
		Transport: &http.Transport{},
	}
	return &Client{
		http:      httpClient,
		baseUrl:   baseUrl,
		logPrefix: logPrefix,
		ua:        ua,
	}
}

type Client struct {
	http      http.Client
	baseUrl   string
	logPrefix string
	ua        string
}

func (client *Client) Get(path string, values url.Values) (*http.Response, error) {
	fullPath := fmt.Sprintf("%v/%v?%v", client.baseUrl, path, values.Encode())
	return client.executeRequest("GET", fullPath, nil)
}

func (client *Client) Post(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *Client) Put(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("PUT", path, payload)
}

func (client *Client) Delete(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("DELETE", path, payload)
}

func (client Client) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	return client.executeRequest(method, client.baseUrl+"/"+path, payloadJson)
}

func (client *Client) executeRequest(method, fullPath string, payloadJson []byte) (*http.Response, error) {
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
