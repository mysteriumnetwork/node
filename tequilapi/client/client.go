package client

import (
	"errors"
	"fmt"
	"net/url"
)

// NewClient returns a new instance of Client
func NewClient(ip string, port int) *Client {
	return &Client{
		http: newHttpClient(
			fmt.Sprintf("http://%s:%d", ip, port),
			"[Tequilapi.Client]",
			"goclient-v0.1",
		),
	}
}

// Client is able perform remote requests to Tequilapi server
type Client struct {
	http httpClientInterface
}

// GetIdentities returns a list of client identities
func (client *Client) GetIdentities() (ids []IdentityDTO, err error) {
	response, err := client.http.Get("identities", url.Values{})
	if err != nil {
		return
	}
	defer response.Body.Close()

	var list IdentityList
	err = parseResponseJson(response, &list)

	return list.Identities, err
}

// NewIdentity creates a new client identity
func (client *Client) NewIdentity(passphrase string) (id IdentityDTO, err error) {
	payload := struct {
		Passphrase string `json:"passphrase"`
	}{
		passphrase,
	}
	response, err := client.http.Post("identities", payload)
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJson(response, &id)
	return id, err
}

// RegisterIdentity registers given identity
func (client *Client) RegisterIdentity(address string) (err error) {
	payload := struct {
		Registered bool `json:"registered"`
	}{
		true,
	}
	response, err := client.http.Put("identities/"+address+"/registration", payload)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return nil
}

// Connect initiates a new connection to a host identified by providerID
func (client *Client) Connect(consumerID, providerID string) (status StatusDTO, err error) {
	payload := struct {
		Identity   string `json:"identity"`
		ProviderID string `json:"providerId"`
	}{
		consumerID,
		providerID,
	}
	response, err := client.http.Put("connection", payload)

	var errorMessage struct {
		Message string `json:"message"`
	}

	if err != nil {
		parseResponseJson(response, &errorMessage)
		err = errors.New(errorMessage.Message)
		return
	}
	defer response.Body.Close()

	err = parseResponseJson(response, &status)
	return status, err
}

// Disconnect terminates current connection
func (client *Client) Disconnect() (err error) {
	response, err := client.http.Delete("connection", nil)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return nil
}

// ConnectionStatistics returns statistics about current connection
func (client *Client) ConnectionStatistics() (StatisticsDTO, error) {
	response, err := client.http.Get("connection/statistics", url.Values{})
	if err != nil {
		return StatisticsDTO{}, err
	}
	defer response.Body.Close()

	var statistics StatisticsDTO
	err = parseResponseJson(response, &statistics)
	return statistics, err
}

// Status returns connection status
func (client *Client) Status() (StatusDTO, error) {
	response, err := client.http.Get("connection", url.Values{})
	if err != nil {
		return StatusDTO{}, err
	}
	defer response.Body.Close()

	var status StatusDTO
	err = parseResponseJson(response, &status)
	return status, err
}

// Proposals returns all available proposals for services
func (client *Client) Proposals() ([]ProposalDTO, error) {
	response, err := client.http.Get("proposals", url.Values{})
	if err != nil {
		return []ProposalDTO{}, err
	}
	defer response.Body.Close()

	var proposals ProposalList
	err = parseResponseJson(response, &proposals)
	return proposals.Proposals, err
}

// GetIP returns public ip
func (client *Client) GetIP() (string, error) {
	response, err := client.http.Get("connection/ip", url.Values{})
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var ipData struct {
		IP string `json:"ip"`
	}
	err = parseResponseJson(response, &ipData)
	return ipData.IP, nil
}

// Unlock allows using identity in following commands
func (client *Client) Unlock(identity, passphrase string) error {
	path := fmt.Sprintf("identities/%s/unlock", identity)
	payload := struct {
		Passphrase string `json:"passphrase"`
	}{
		passphrase,
	}

	response, err := client.http.Put(path, payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}
