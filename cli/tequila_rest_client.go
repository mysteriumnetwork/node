package cli

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"github.com/mysterium/node/identity"
)

func NewTequilaClient() *TequilaClient {
	return &TequilaClient{
		http: NewHttpClient(
			"http://127.0.0.1:4050",
			"[Tequilla.api.client]",
			"goclient-v0.1",
		),
	}
}

type TequilaClient struct {
	http HttpClientInterface
}

type StatusDto struct {
	Status    string `json:"status"`
	SessionId string `json:"sessionId"`
}

type identityDto struct {
	Address string `json:"id"`
}

type identityList struct {
	Identities []identityDto `json:"identities"`
}

func (client *TequilaClient) GetIdentities() (identities []identity.Identity, err error) {
	response, err := client.http.Get("identities", url.Values{})
	if err != nil {
		return
	}

	defer response.Body.Close()

	var list identityList
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

func (client *TequilaClient) NewIdentity() (id identity.Identity, err error) {
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

func (client *TequilaClient) Register() (id identity.Identity, err error) {
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

func (client *TequilaClient) Connect(id identity.Identity, providerId identity.Identity) (err error) {
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

func (client *TequilaClient) Disconnect() (err error) {
	response, err := client.http.Delete("connection", nil)
	if err != nil {
		return
	}

	response.Body.Close()

	return nil
}

func (client *TequilaClient) Status() (status StatusDto, err error) {
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
