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

const IpifyAPIURL = "https://api.ipify.org/"
const IpifiAPICLient = "goclient-v0.1"
const IpifiAPILogPrefix = "[ipify.api] "

func NewResolver() Resolver {
	return NewResolverWithTimeout(60 * time.Second)
}

func NewResolverWithTimeout(timeout time.Duration) Resolver {
	return &clientRest{
		httpClient: http.Client{
			Timeout: timeout,
		},
	}
}

type clientRest struct {
	httpClient http.Client
}

func (client *clientRest) GetPublicIP() (string, error) {
	var ipResponse IPResponse

	request, err := http.NewRequest("GET", IpifyAPIURL+"/?format=json", nil)
	request.Header.Set("User-Agent", IpifiAPICLient)
	request.Header.Set("Accept", "application/json")
	if err != nil {
		log.Critical(IpifiAPILogPrefix, err)
		return "", err
	}

	err = client.doRequest(request, &ipResponse)
	if err != nil {
		return "", err
	}

	log.Info(IpifiAPILogPrefix, "IP detected: ", ipResponse.IP)
	return ipResponse.IP, nil
}

func (client *clientRest) GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	defer conn.Close()
	if err != nil {
		return "", err
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Info("[Detect Outbound IP] ", "IP detected: ", localAddr.IP.String())
	return localAddr.IP.String(), nil
}

func (client *clientRest) doRequest(request *http.Request, responseDto interface{}) error {
	response, err := client.httpClient.Do(request)
	if err != nil {
		log.Error(IpifiAPILogPrefix, err)
		return err
	}
	defer response.Body.Close()

	err = parseResponseError(response)
	if err != nil {
		log.Error(IpifiAPILogPrefix, err)
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
