package client

import (
	"encoding/json"
	"fmt"
	"github.com/mysterium/node/identity"
	"io/ioutil"
	"net/http"
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

// Client is able perform remote requests to Teuilapi server
type Client struct {
	http httpClientInterface
}

// StatusDto holds connection status and session id
type StatusDto struct {
	Status    string `json:"status"`
	SessionId string `json:"sessionId"`
}

// IdentityDto holds identity address
type IdentityDto struct {
	Address string `json:"id"`
}

// IdentityList holds returned list of identities
type IdentityList struct {
	Identities []IdentityDto `json:"identities"`
}

// GetIdentities returns a list of client identities
func (client *Client) GetIdentities() (identities []identity.Identity, err error) {
	response, err := client.http.Get("identities", url.Values{})
	if err != nil {
		return
	}

	defer response.Body.Close()

	var list IdentityList
	err = parseResponseJson(response, &list)
	if err != nil {
		return
	}

	identities = make([]identity.Identity, len(list.Identities))
	for i, id := range list.Identities {
		identities[i] = identity.FromAddress(id.Address)
	}

	return
}

// NewIdentity create a new client identity
func (client *Client) NewIdentity() (id identity.Identity, err error) {
	payload := struct {
		Password string `json:"password"`
	}{
		"",
	}
	response, err := client.http.Post("identities", payload)
	if err != nil {
		return
	}

	defer response.Body.Close()

	var idDto struct {
		Id string `json:"id"`
	}
	err = parseResponseJson(response, &idDto)
	if err != nil {
		return
	}

	return identity.FromAddress(idDto.Id), nil
}

// Register registers given identity to discovery service
func (client *Client) Register() (id identity.Identity, err error) {
	payload := struct {
		Id string `json:"id"`
	}{
		"",
	}
	response, err := client.http.Put("identities/"+id.Address+"/registration", payload)
	if err != nil {
		return
	}

	defer response.Body.Close()

	var idDto struct {
		Id string `json:"id"`
	}
	err = parseResponseJson(response, &idDto)
	if err != nil {
		return
	}

	return identity.FromAddress(idDto.Id), nil
}

// Connect initiates a new connection to a host identified by providerId
func (client *Client) Connect(id identity.Identity, providerId identity.Identity) (err error) {
	payload := struct {
		Identity string `json:"identity"`
		NodeKey  string `json:"nodeKey"`
	}{
		id.Address,
		providerId.Address,
	}
	response, err := client.http.Put("connection", payload)
	if err != nil {
		return
	}

	response.Body.Close()

	return nil
}

// Disconnect terminates current connection
func (client *Client) Disconnect() (err error) {
	response, err := client.http.Delete("connection", nil)
	if err != nil {
		return
	}

	response.Body.Close()

	return nil
}

// Status returns connection status
func (client *Client) Status() (status StatusDto, err error) {
	response, err := client.http.Get("connection", url.Values{})
	defer response.Body.Close()
	if err != nil {
		return
	}

	err = parseResponseJson(response, &status)
	if err != nil {
		return
	}

	return status, nil
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
