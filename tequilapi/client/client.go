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
	"fmt"
	"net/http"
	"net/url"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/pkg/errors"
)

// NewClient returns a new instance of Client
func NewClient(ip string, port int) *Client {
	return &Client{
		http: newHTTPClient(
			fmt.Sprintf("http://%s:%d", ip, port),
			"goclient-v0.1",
		),
	}
}

// Client is able perform remote requests to Tequilapi server
type Client struct {
	http httpClientInterface
}

// GetIdentities returns a list of client identities
func (client *Client) GetIdentities() (ids []contract.IdentityRefDTO, err error) {
	response, err := client.http.Get("identities", url.Values{})
	if err != nil {
		return
	}
	defer response.Body.Close()

	var list contract.ListIdentitiesResponse
	err = parseResponseJSON(response, &list)

	return list.Identities, err
}

// NewIdentity creates a new client identity
func (client *Client) NewIdentity(passphrase string) (id contract.IdentityRefDTO, err error) {
	response, err := client.http.Post("identities", contract.IdentityCreateRequest{Passphrase: &passphrase})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// CurrentIdentity unlocks and returns the last used, new or first identity
func (client *Client) CurrentIdentity(identity, passphrase string) (id contract.IdentityRefDTO, err error) {
	response, err := client.http.Put("identities/current", contract.IdentityCurrentRequest{
		Address:    &identity,
		Passphrase: &passphrase,
	})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// Identity returns identity status with current balance
func (client *Client) Identity(identityAddress string) (contract.IdentityDTO, error) {
	path := fmt.Sprintf("identities/%s", identityAddress)

	response, err := client.http.Get(path, nil)
	if err != nil {
		return contract.IdentityDTO{}, err
	}
	defer response.Body.Close()

	res := contract.IdentityDTO{}
	err = parseResponseJSON(response, &res)
	return res, err
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

// GetTransactorFees returns the transactor fees
func (client *Client) GetTransactorFees() (Fees, error) {
	fees := Fees{}

	res, err := client.http.Get("transactor/fees", nil)
	if err != nil {
		return fees, err
	}
	defer res.Body.Close()

	err = parseResponseJSON(res, &fees)
	return fees, err
}

// RegisterIdentity registers identity
func (client *Client) RegisterIdentity(address, beneficiary string, stake, fee uint64) error {
	payload := registry.IdentityRegistrationRequestDTO{
		Stake:       stake,
		Fee:         fee,
		Beneficiary: beneficiary,
	}

	response, err := client.http.Post("identities/"+address+"/register", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected 202 got %v", response.StatusCode)
	}

	return nil
}

// TopUp tops up identity
func (client *Client) TopUp(identity string) error {
	payload := registry.TopUpRequest{
		Identity: identity,
	}

	response, err := client.http.Post("transactor/topup", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected 202 got %v", response.StatusCode)
	}

	return nil
}

// ConnectionCreate initiates a new connection to a host identified by providerID
func (client *Client) ConnectionCreate(consumerID, providerID, accountantID, serviceType string, options contract.ConnectOptions) (status contract.ConnectionStatusDTO, err error) {
	response, err := client.http.Put("connection", contract.ConnectionCreateRequest{
		ConsumerID:     consumerID,
		ProviderID:     providerID,
		AccountantID:   accountantID,
		ServiceType:    serviceType,
		ConnectOptions: options,
	})
	if err != nil {
		return contract.ConnectionStatusDTO{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionDestroy terminates current connection
func (client *Client) ConnectionDestroy() (err error) {
	response, err := client.http.Delete("connection", nil)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return nil
}

// ConnectionStatistics returns statistics about current connection
func (client *Client) ConnectionStatistics() (contract.ConnectionStatisticsDTO, error) {
	response, err := client.http.Get("connection/statistics", url.Values{})
	if err != nil {
		return contract.ConnectionStatisticsDTO{}, err
	}
	defer response.Body.Close()

	var statistics contract.ConnectionStatisticsDTO
	err = parseResponseJSON(response, &statistics)
	return statistics, err
}

// ConnectionStatus returns connection status
func (client *Client) ConnectionStatus() (contract.ConnectionStatusDTO, error) {
	response, err := client.http.Get("connection", url.Values{})
	if err != nil {
		return contract.ConnectionStatusDTO{}, err
	}
	defer response.Body.Close()

	var status contract.ConnectionStatusDTO
	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionIP returns public ip
func (client *Client) ConnectionIP() (string, error) {
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

// ConnectionLocation returns current location
func (client *Client) ConnectionLocation() (location LocationDTO, err error) {
	response, err := client.http.Get("connection/location", url.Values{})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
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

// OriginLocation returns original location
func (client *Client) OriginLocation() (location LocationDTO, err error) {
	response, err := client.http.Get("location", url.Values{})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
}

// ProposalsByType fetches proposals by given type
func (client *Client) ProposalsByType(serviceType string) ([]contract.ProposalDTO, error) {
	queryParams := url.Values{}
	queryParams.Add("service_type", serviceType)
	return client.proposals(queryParams)
}

// Proposals returns all available proposals for services
func (client *Client) Proposals() ([]contract.ProposalDTO, error) {
	return client.proposals(url.Values{})
}

func (client *Client) proposals(query url.Values) ([]contract.ProposalDTO, error) {
	response, err := client.http.Get("proposals", query)
	if err != nil {
		return []contract.ProposalDTO{}, err
	}
	defer response.Body.Close()

	var proposals contract.ListProposalsResponse
	err = parseResponseJSON(response, &proposals)
	return proposals.Proposals, err
}

// ProposalsByPrice returns all available proposals within the given price range
func (client *Client) ProposalsByPrice(lowerTime, upperTime, lowerGB, upperGB uint64) ([]contract.ProposalDTO, error) {
	values := url.Values{}
	values.Add("upper_time_price_bound", fmt.Sprintf("%v", upperTime))
	values.Add("lower_time_price_bound", fmt.Sprintf("%v", lowerTime))
	values.Add("upper_gb_price_bound", fmt.Sprintf("%v", upperGB))
	values.Add("lower_gb_price_bound", fmt.Sprintf("%v", lowerGB))
	return client.proposals(values)
}

// Unlock allows using identity in following commands
func (client *Client) Unlock(identity, passphrase string) error {
	path := fmt.Sprintf("identities/%s/unlock", identity)

	response, err := client.http.Put(path, contract.IdentityUnlockRequest{Passphrase: &passphrase})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// Payout registers payout address for identity
func (client *Client) Payout(identity, ethAddress string) error {
	path := fmt.Sprintf("identities/%s/payout", identity)
	payload := struct {
		EthAddress string `json:"eth_address"`
	}{
		ethAddress,
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

// ConnectionSessions returns all sessions from history
func (client *Client) ConnectionSessions() (ConnectionSessionListDTO, error) {
	sessions := ConnectionSessionListDTO{}
	response, err := client.http.Get("connection-sessions", url.Values{})
	if err != nil {
		return sessions, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &sessions)
	return sessions, err
}

// ConnectionSessionsByType returns sessions from history filtered by type
func (client *Client) ConnectionSessionsByType(serviceType string) (ConnectionSessionListDTO, error) {
	sessions, err := client.ConnectionSessions()
	sessions = filterSessionsByType(serviceType, sessions)
	return sessions, err
}

// ConnectionSessionsByStatus returns sessions from history filtered by their status
func (client *Client) ConnectionSessionsByStatus(status string) (ConnectionSessionListDTO, error) {
	sessions, err := client.ConnectionSessions()
	sessions = filterSessionsByStatus(status, sessions)
	return sessions, err
}

// Services returns all running services
func (client *Client) Services() (services ServiceListDTO, err error) {
	response, err := client.http.Get("services", url.Values{})
	if err != nil {
		return services, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &services)
	return services, err
}

// Service returns a service information by the requested id
func (client *Client) Service(id string) (service ServiceInfoDTO, err error) {
	response, err := client.http.Get("services/"+id, url.Values{})
	if err != nil {
		return service, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &service)
	return service, err
}

// ServiceStart starts an instance of the service.
func (client *Client) ServiceStart(request contract.ServiceStartRequest) (service ServiceInfoDTO, err error) {
	response, err := client.http.Post("services", request)
	if err != nil {
		return service, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &service)
	return service, err
}

// ServiceStop stops the running service instance by the requested id.
func (client *Client) ServiceStop(id string) error {
	path := fmt.Sprintf("services/%s", id)
	response, err := client.http.Delete(path, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// NATStatus returns status of NAT traversal
func (client *Client) NATStatus() (NATStatusDTO, error) {
	status := NATStatusDTO{}

	response, err := client.http.Get("nat/status", nil)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ServiceSessions returns all currently running sessions
func (client *Client) ServiceSessions() (ServiceSessionListDTO, error) {
	sessions := ServiceSessionListDTO{}
	response, err := client.http.Get("service-sessions", url.Values{})
	if err != nil {
		return sessions, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &sessions)
	return sessions, err
}

// filterSessionsByType removes all sessions of irrelevant types
func filterSessionsByType(serviceType string, sessions ConnectionSessionListDTO) ConnectionSessionListDTO {
	matches := 0
	for _, s := range sessions.Sessions {
		if s.ServiceType == serviceType {
			sessions.Sessions[matches] = s
			matches++
		}
	}
	sessions.Sessions = sessions.Sessions[:matches]
	return sessions
}

// filterSessionsByStatus removes all sessions with non matching status
func filterSessionsByStatus(status string, sessions ConnectionSessionListDTO) ConnectionSessionListDTO {
	matches := 0
	for _, s := range sessions.Sessions {
		if s.Status == status {
			sessions.Sessions[matches] = s
			matches++
		}
	}
	sessions.Sessions = sessions.Sessions[:matches]
	return sessions
}

// Settle requests the settling of accountant promises
func (client *Client) Settle(providerID, accountantID identity.Identity, waitForBlockchain bool) error {
	settleRequest := SettleRequest{
		ProviderID:   providerID.Address,
		AccountantID: accountantID.Address,
	}

	path := "transactor/settle/"
	if waitForBlockchain {
		path += "sync"
	} else {
		path += "async"
	}

	response, err := client.http.Post(path, settleRequest)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not settle promise")
	}
	return nil
}

// SetBeneficiary set new beneficiary address for the provided identity.
func (client *Client) SetBeneficiary(address, beneficiary string) error {
	payload := SetBeneficiaryRequest{Beneficiary: beneficiary}
	response, err := client.http.Post("identities/"+address+"/beneficiary", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected 202 got %v", response.StatusCode)
	}

	return nil
}

// Beneficiary gets beneficiary address for the provided identity.
func (client *Client) Beneficiary(address string) (res contract.IdentityBeneficiaryResponce, err error) {
	response, err := client.http.Get("identities/"+address+"/beneficiary", nil)
	if err != nil {
		return contract.IdentityBeneficiaryResponce{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	return res, err
}
