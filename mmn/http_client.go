package mmn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

type httpClientInterface interface {
	Get(path string, values url.Values) (*http.Response, error)
	Post(path string, payload interface{}) (*http.Response, error)
}

type httpRequestInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpClient struct {
	http      httpRequestInterface
	baseURL   string
	logPrefix string
	ua        string
}

// NewMMNClient returns a new instance of Client
func NewMMNClient(url string) *Client {
	return &Client{
		http: newHTTPClient(
			url,
			"[MMN.Client] ",
			"goclient-v0.1",
		),
	}
}

func (c *Client) RegisterNode(information NodeInformation) error {
	response, err := c.http.Post("node", information)
	if err != nil {
		return err
	}

	print(parseResponseJSON(response, nil))

	return nil
}

// Client is able perform remote requests to Tequilapi server
type Client struct {
	http httpClientInterface
}

func newHTTPClient(baseURL string, logPrefix string, ua string) *httpClient {
	return &httpClient{
		http: &http.Client{
			Transport: &http.Transport{},
			Timeout:   10 * time.Second,
		},
		baseURL:   baseURL,
		logPrefix: logPrefix,
		ua:        ua,
	}
}
func (client httpClient) doPayloadRequest(method, path string, payload interface{}) (*http.Response, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}

	return client.executeRequest(method, client.baseURL+"/"+path, payloadJSON)
}

func (client *httpClient) executeRequest(method, fullPath string, payloadJSON []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fullPath, bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Critical(client.logPrefix, err)
		return nil, err
	}
	request.Header.Set("User-Agent", client.ua)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

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

func (client *httpClient) Post(path string, payload interface{}) (*http.Response, error) {
	return client.doPayloadRequest("POST", path, payload)
}

func (client *httpClient) Get(path string, values url.Values) (*http.Response, error) {
	basePath := fmt.Sprintf("%v/%v", client.baseURL, path)

	var fullPath string
	params := values.Encode()
	if params == "" {
		fullPath = basePath
	} else {
		fullPath = fmt.Sprintf("%v?%v", basePath, params)
	}
	return client.executeRequest("GET", fullPath, nil)
}

type errorBody struct {
	Message string `json:"message"`
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		//sometimes we can get json message with single "message" field which represents error - try to get that
		var parsedBody errorBody
		var message string
		err := parseResponseJSON(response, &parsedBody)
		if err != nil {
			message = err.Error()
		} else {
			message = parsedBody.Message
		}
		// TODO these errors are ugly long and hard to check against - consider return error structs or specific error constants
		return errors.Errorf("server response invalid: %s (%s). Possible error: %s", response.Status, response.Request.URL, message)
	}

	return nil
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

	return response.Body.Close()
}
