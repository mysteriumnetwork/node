package client

import (
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
func (client *Client) GetIdentities() (ids []IdentityDto, err error) {
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
func (client *Client) NewIdentity(password string) (id IdentityDto, err error) {
	payload := struct {
		Password string `json:"password"`
	}{
		password,
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

// Connect initiates a new connection to a host identified by providerId
func (client *Client) Connect(consumerId, providerId string) (status StatusDto, err error) {
	payload := struct {
		Identity string `json:"identity"`
		NodeKey  string `json:"nodeKey"`
	}{
		consumerId,
		providerId,
	}
	response, err := client.http.Put("connection", payload)
	if err != nil {
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

// Status returns connection status
func (client *Client) Status() (status StatusDto, err error) {
	response, err := client.http.Get("connection", url.Values{})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJson(response, &status)
	return status, err
}

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
