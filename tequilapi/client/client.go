/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package client

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/mysteriumnetwork/node/tequilapi/endpoints"
)

// NewClient returns a new instance of Client
func NewClient(ip string, port int) *Client {
	return &Client{
		http: newHTTPClient(
			fmt.Sprintf("http://%s:%d", ip, port),
			"[Tequilapi.Client] ",
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
	err = parseResponseJSON(response, &list)

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

	err = parseResponseJSON(response, &id)
	return id, err
}

// IdentityRegistrationStatus returns information of identity needed to register it on blockchain
func (client *Client) IdentityRegistrationStatus(address string) (RegistrationDataDTO, error) {
	response, err := client.http.Get("identities/"+address+"/registration", url.Values{})
	if err != nil {
		return RegistrationDataDTO{}, err
	}
	defer response.Body.Close()

	status := RegistrationDataDTO{}
	err = parseResponseJSON(response, &status)
	return status, err
}

// Connect initiates a new connection to a host identified by providerID
func (client *Client) Connect(consumerID, providerID, serviceType string, options endpoints.ConnectOptions) (status StatusDTO, err error) {
	payload := struct {
		Identity    string                   `json:"consumerId"`
		ProviderID  string                   `json:"providerId"`
		ServiceType string                   `json:"serviceType"`
		Options     endpoints.ConnectOptions `json:"connectOptions"`
	}{
		Identity:    consumerID,
		ProviderID:  providerID,
		ServiceType: serviceType,
		Options:     options,
	}
	response, err := client.http.Put("connection", payload)

	var errorMessage struct {
		Message string `json:"message"`
	}

	if err != nil {
		err = parseResponseJSON(response, &errorMessage)
		if err != nil {
			return
		}

		err = errors.New(errorMessage.Message)
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
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
	err = parseResponseJSON(response, &statistics)
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
	err = parseResponseJSON(response, &status)
	return status, err
}

// Healthcheck returns a healthcheck info
func (client *Client) Healthcheck() (healthcheck HealthcheckDTO, err error) {
	response, err := client.http.Get("healthcheck", url.Values{})
	if err != nil {
		return
	}

	defer response.Body.Close()
	err = parseResponseJSON(response, &healthcheck)
	return healthcheck, err
}

// Proposals returns all available proposals for services
func (client *Client) Proposals() ([]ProposalDTO, error) {
	response, err := client.http.Get("proposals", url.Values{})
	if err != nil {
		return []ProposalDTO{}, err
	}
	defer response.Body.Close()

	var proposals ProposalList
	err = parseResponseJSON(response, &proposals)
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
	err = parseResponseJSON(response, &ipData)
	return ipData.IP, err
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

// Stop kills mysterium client
func (client *Client) Stop() error {
	emptyPayload := struct{}{}
	response, err := client.http.Post("/stop", emptyPayload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// GetSessions returns all sessions from history
func (client *Client) GetSessions() (endpoints.SessionsDTO, error) {
	sessions := endpoints.SessionsDTO{}
	response, err := client.http.Get("sessions", url.Values{})
	if err != nil {
		return sessions, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &sessions)
	return sessions, err
}
