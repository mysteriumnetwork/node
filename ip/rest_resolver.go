package ip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/cihub/seelog"
	"net"
	"time"
)

const ipifyAPIURL = "https://api.ipify.org/"
const ipifyAPIClient = "goclient-v0.1"
const ipifyAPILogPrefix = "[ipify.api] "

func NewResolver() Resolver {
	return NewResolverWithTimeout(1 * time.Minute)
}

func NewResolverWithTimeout(timeout time.Duration) Resolver {
	return &clientRest{
		httpClient: http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				//dont cache tcp connections - first requests after state change (direct -> tunneled and vice versa) will always fail
				//as stale tcp states are not closed after switch. Probably some kind of CloseIdleConnections will help in the future
				DisableKeepAlives: true,
			},
		},
	}
}

type clientRest struct {
	httpClient http.Client
}

func (client *clientRest) GetPublicIP() (string, error) {
	var ipResponse IPResponse

	request, err := http.NewRequest("GET", ipifyAPIURL+"/?format=json", nil)
	request.Header.Set("User-Agent", ipifyAPIClient)
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(ipifyAPILogPrefix, err)
		return "", err
	}

	err = client.doRequest(request, &ipResponse)
	if err != nil {
		return "", err
	}

	log.Info(ipifyAPILogPrefix, "IP detected: ", ipResponse.IP)
	return ipResponse.IP, nil
}

func (client *clientRest) GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Info("[Detect Outbound IP] ", "IP detected: ", localAddr.IP.String())
	return localAddr.IP.String(), nil
}

func (client *clientRest) doRequest(request *http.Request, responseDto interface{}) error {
	response, err := client.httpClient.Do(request)
	if err != nil {
		log.Error(ipifyAPILogPrefix, err)
		return err
	}
	defer response.Body.Close()

	err = parseResponseError(response)
	if err != nil {
		log.Error(ipifyAPILogPrefix, err)
		return err
	}

	return parseResponseJson(response, &responseDto)
}

func parseResponseJson(response *http.Response, dto interface{}) error {
	responseJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(responseJson, dto)
	if err != nil {
		return err
	}

	return nil
}

func parseResponseError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("server response invalid: %s (%s)", response.Status, response.Request.URL)
	}

	return nil
}
