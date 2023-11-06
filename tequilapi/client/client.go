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
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/payments/exchange"
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

// AuthAuthenticate authenticates user and issues auth token
func (client *Client) AuthAuthenticate(request contract.AuthRequest) (res contract.AuthResponse, err error) {
	response, err := client.http.Post("/auth/authenticate", request)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	if err != nil {
		return res, err
	}

	client.http.SetToken(res.Token)
	return res, nil
}

// AuthLogin authenticates user and sets cookie with issued auth token
func (client *Client) AuthLogin(request contract.AuthRequest) (res contract.AuthResponse, err error) {
	response, err := client.http.Post("/auth/login", request)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	if err != nil {
		return res, err
	}

	client.http.SetToken(res.Token)
	return res, nil
}

// AuthLogout Clears authentication cookie
func (client *Client) AuthLogout() error {
	response, err := client.http.Delete("/auth/logout", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// AuthChangePassword changes user password
func (client *Client) AuthChangePassword(request contract.ChangePasswordRequest) error {
	response, err := client.http.Put("/auth/password", request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// ImportIdentity sends a request to import a given identity.
func (client *Client) ImportIdentity(blob []byte, passphrase string, setDefault bool) (id contract.IdentityRefDTO, err error) {
	response, err := client.http.Post("identities-import", contract.IdentityImportRequest{
		Data:              blob,
		CurrentPassphrase: passphrase,
		SetDefault:        setDefault,
	})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
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

// BalanceRefresh forces a balance refresh if possible and returns the latest balance.
func (client *Client) BalanceRefresh(identityAddress string) (b contract.BalanceDTO, err error) {
	path := fmt.Sprintf("identities/%s/balance/refresh", identityAddress)

	response, err := client.http.Put(path, nil)
	if err != nil {
		return b, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &b)
	return b, err
}

// Identity returns identity status with cached balance
func (client *Client) Identity(identityAddress string) (id contract.IdentityDTO, err error) {
	path := fmt.Sprintf("identities/%s", identityAddress)

	response, err := client.http.Get(path, nil)
	if err != nil {
		return id, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// IdentityRegistrationStatus returns information of identity needed to register it on blockchain
func (client *Client) IdentityRegistrationStatus(address string) (contract.IdentityRegistrationResponse, error) {
	response, err := client.http.Get("identities/"+address+"/registration", url.Values{})
	if err != nil {
		return contract.IdentityRegistrationResponse{}, err
	}
	defer response.Body.Close()

	status := contract.IdentityRegistrationResponse{}
	err = parseResponseJSON(response, &status)
	return status, err
}

// GetTransactorFees returns the transactor fees
func (client *Client) GetTransactorFees() (contract.FeesDTO, error) {
	fees := contract.FeesDTO{}

	res, err := client.http.Get("transactor/fees", nil)
	if err != nil {
		return fees, err
	}
	defer res.Body.Close()

	err = parseResponseJSON(res, &fees)
	return fees, err
}

// RegisterIdentity registers identity
func (client *Client) RegisterIdentity(address, beneficiary string, token *string) error {
	payload := contract.IdentityRegisterRequest{
		ReferralToken: token,
		Beneficiary:   beneficiary,
	}

	response, err := client.http.Post("identities/"+address+"/register", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusAccepted:
	default:
		return fmt.Errorf("expected 200 or 202 got %v", response.StatusCode)
	}

	return nil
}

// GetRegistrationPaymentStatus returns the registration payment status
func (client *Client) GetRegistrationPaymentStatus(identity string) (contract.RegistrationPaymentResponse, error) {
	resp := contract.RegistrationPaymentResponse{}

	res, err := client.http.Get(fmt.Sprintf("v2/identities/%s/registration-payment", identity), nil)
	if err != nil {
		return resp, err
	}
	defer res.Body.Close()

	err = parseResponseJSON(res, &resp)
	return resp, err
}

// ConnectionCreate initiates a new connection to a host identified by providerID
func (client *Client) ConnectionCreate(consumerID, providerID, hermesID, serviceType string, options contract.ConnectOptions) (status contract.ConnectionInfoDTO, err error) {
	response, err := client.http.Put("connection", contract.ConnectionCreateRequest{
		ConsumerID:     consumerID,
		ProviderID:     providerID,
		HermesID:       hermesID,
		ServiceType:    serviceType,
		ConnectOptions: options,
		Filter: contract.ConnectionCreateFilter{
			IncludeMonitoringFailed: true,
		},
	})
	if err != nil {
		return contract.ConnectionInfoDTO{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// SmartConnectionCreate initiates a new connection to a host identified by filter
func (client *Client) SmartConnectionCreate(consumerID, hermesID, serviceType string, filter contract.ConnectionCreateFilter, options contract.ConnectOptions) (status contract.ConnectionInfoDTO, err error) {
	response, err := client.http.Put("connection", contract.ConnectionCreateRequest{
		ConsumerID:     consumerID,
		Filter:         filter,
		HermesID:       hermesID,
		ServiceType:    serviceType,
		ConnectOptions: options,
	})
	if err != nil {
		return contract.ConnectionInfoDTO{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionDestroy terminates current connection
func (client *Client) ConnectionDestroy(port int) (err error) {
	url := fmt.Sprintf("connection?%s", url.Values{"id": []string{strconv.Itoa(port)}}.Encode())
	response, err := client.http.Delete(url, nil)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return nil
}

// ConnectionStatistics returns statistics about current connection
func (client *Client) ConnectionStatistics(sessionID ...string) (statistics contract.ConnectionStatisticsDTO, err error) {
	response, err := client.http.Get("connection/statistics", url.Values{
		"id": sessionID,
	})
	if err != nil {
		return statistics, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &statistics)
	return statistics, err
}

// ConnectionTraffic returns traffic information about current connection
func (client *Client) ConnectionTraffic(sessionID ...string) (traffic contract.ConnectionTrafficDTO, err error) {
	response, err := client.http.Get("connection/traffic", url.Values{
		"id": sessionID,
	})
	if err != nil {
		return traffic, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &traffic)
	return traffic, err
}

// ConnectionStatus returns connection status
func (client *Client) ConnectionStatus(port int) (status contract.ConnectionInfoDTO, err error) {
	response, err := client.http.Get("connection", url.Values{"id": []string{strconv.Itoa(port)}})
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionIP returns public ip
func (client *Client) ConnectionIP() (ip contract.IPDTO, err error) {
	response, err := client.http.Get("connection/ip", url.Values{})
	if err != nil {
		return ip, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &ip)
	return ip, err
}

// ProxyIP returns public ip of the proxy.
func (client *Client) ProxyIP(proxyPort int) (ip contract.IPDTO, err error) {
	response, err := client.http.Get("connection/proxy/ip", url.Values{"port": []string{strconv.Itoa(proxyPort)}})
	if err != nil {
		return ip, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &ip)
	return ip, err
}

// ProxyLocation returns proxy location.
func (client *Client) ProxyLocation(proxyPort int) (location contract.LocationDTO, err error) {
	response, err := client.http.Get("connection/proxy/location", url.Values{"port": []string{strconv.Itoa(proxyPort)}})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
}

// ConnectionLocation returns current location
func (client *Client) ConnectionLocation() (location contract.LocationDTO, err error) {
	response, err := client.http.Get("connection/location", url.Values{})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
}

// Healthcheck returns a healthcheck info
func (client *Client) Healthcheck() (healthcheck contract.HealthCheckDTO, err error) {
	response, err := client.http.Get("healthcheck", url.Values{})
	if err != nil {
		return
	}

	defer response.Body.Close()
	err = parseResponseJSON(response, &healthcheck)
	return healthcheck, err
}

// OriginLocation returns original location
func (client *Client) OriginLocation() (location contract.LocationDTO, err error) {
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

// ProposalsByTypeWithWhitelisting fetches proposals by given type with all whitelisting options.
func (client *Client) ProposalsByTypeWithWhitelisting(serviceType string) ([]contract.ProposalDTO, error) {
	queryParams := url.Values{}
	queryParams.Add("service_type", serviceType)
	queryParams.Add("access_policy", "all")
	return client.proposals(queryParams)
}

// ProposalsByLocationAndService fetches proposals by given service and node location types.
func (client *Client) ProposalsByLocationAndService(serviceType, locationType, locationCountry string) ([]contract.ProposalDTO, error) {
	queryParams := url.Values{}
	queryParams.Add("service_type", serviceType)
	queryParams.Add("ip_type", locationType)
	queryParams.Add("location_country", locationCountry)
	return client.proposals(queryParams)
}

// Proposals returns all available proposals for services
func (client *Client) Proposals() ([]contract.ProposalDTO, error) {
	return client.proposals(url.Values{})
}

// ProposalsNATCompatible returns proposals for services which we can connect to
func (client *Client) ProposalsNATCompatible() ([]contract.ProposalDTO, error) {
	queryParams := url.Values{}
	queryParams.Add("nat_compatibility", contract.AutoNATType)
	return client.proposals(queryParams)
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

// Unlock allows using identity in following commands
func (client *Client) Unlock(identity, passphrase string) error {
	payload := contract.IdentityUnlockRequest{
		Passphrase: &passphrase,
	}

	path := fmt.Sprintf("identities/%s/unlock", identity)
	response, err := client.http.Put(path, payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// SetBeneficiaryAsync store beneficiary address locally for identity.
func (client *Client) SetBeneficiaryAsync(identity, ethAddress string) error {
	path := fmt.Sprintf("identities/%s/beneficiary-async", identity)
	payload := contract.BeneficiaryAddressRequest{
		Address: ethAddress,
	}

	response, err := client.http.Post(path, payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to save")
	}

	return nil
}

// GetBeneficiaryAsync gets locally saved beneficiary address.
func (client *Client) GetBeneficiaryAsync(identity string) (contract.BeneficiaryAddressRequest, error) {
	path := fmt.Sprintf("identities/%s/beneficiary-async", identity)
	res := contract.BeneficiaryAddressRequest{}
	response, err := client.http.Get(path, nil)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	return res, err
}

// Stop kills mysterium client
func (client *Client) Stop() error {
	emptyPayload := struct{}{}
	response, err := client.http.Post("stop", emptyPayload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// Sessions returns all sessions from history
func (client *Client) Sessions() (sessions contract.SessionListResponse, err error) {
	response, err := client.http.Get("sessions", url.Values{})
	if err != nil {
		return sessions, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &sessions)
	return sessions, err
}

// SessionsByServiceType returns sessions from history filtered by type
func (client *Client) SessionsByServiceType(serviceType string) (contract.SessionListResponse, error) {
	sessions, err := client.Sessions()
	sessions = filterSessionsByType(serviceType, sessions)
	return sessions, err
}

// SessionsByStatus returns sessions from history filtered by their status
func (client *Client) SessionsByStatus(status string) (contract.SessionListResponse, error) {
	sessions, err := client.Sessions()
	sessions = filterSessionsByStatus(status, sessions)
	return sessions, err
}

// Services returns all running services
func (client *Client) Services() (services contract.ServiceListResponse, err error) {
	response, err := client.http.Get("services", url.Values{})
	if err != nil {
		return services, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &services)
	return services, err
}

// Service returns a service information by the requested id
func (client *Client) Service(id string) (service contract.ServiceInfoDTO, err error) {
	response, err := client.http.Get("services/"+id, url.Values{})
	if err != nil {
		return service, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &service)
	return service, err
}

// ServiceStart starts an instance of the service.
func (client *Client) ServiceStart(request contract.ServiceStartRequest) (service contract.ServiceInfoDTO, err error) {
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
func (client *Client) NATStatus() (status contract.NodeStatusResponse, err error) {
	response, err := client.http.Get("node/monitoring-status", nil)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// NATType returns type of NAT in sense of traversal capabilities
func (client *Client) NATType() (status contract.NATTypeDTO, err error) {
	response, err := client.http.Get("nat/type", nil)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// filterSessionsByType removes all sessions of irrelevant types
func filterSessionsByType(serviceType string, sessions contract.SessionListResponse) contract.SessionListResponse {
	matches := 0
	for _, s := range sessions.Items {
		if s.ServiceType == serviceType {
			sessions.Items[matches] = s
			matches++
		}
	}
	sessions.Items = sessions.Items[:matches]
	return sessions
}

// filterSessionsByStatus removes all sessions with non matching status
func filterSessionsByStatus(status string, sessions contract.SessionListResponse) contract.SessionListResponse {
	matches := 0
	for _, s := range sessions.Items {
		if s.Status == status {
			sessions.Items[matches] = s
			matches++
		}
	}
	sessions.Items = sessions.Items[:matches]
	return sessions
}

// Withdraw requests the withdrawal of money from l2 to l1 of hermes promises
func (client *Client) Withdraw(providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int, fromChainID, toChainID int64) error {
	withdrawRequest := contract.WithdrawRequest{
		ProviderID:  providerID.Address,
		HermesID:    hermesID.Hex(),
		Beneficiary: beneficiary.Hex(),
		FromChainID: fromChainID,
		ToChainID:   toChainID,
	}

	if amount != nil {
		withdrawRequest.Amount = amount.String()
	}

	path := "transactor/settle/withdraw"

	response, err := client.http.Post(path, withdrawRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not withdraw")
	}
	return nil
}

// Settle requests the settling of hermes promises
func (client *Client) Settle(providerID identity.Identity, hermesIDs []common.Address, waitForBlockchain bool) error {
	settleRequest := contract.SettleRequest{
		ProviderID: providerID.Address,
		HermesIDs:  hermesIDs,
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
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not settle promise")
	}
	return nil
}

// SettleIntoStake requests the settling of accountant promises into a stake increase
func (client *Client) SettleIntoStake(providerID, hermesID identity.Identity, waitForBlockchain bool) error {
	settleRequest := contract.SettleRequest{
		ProviderID: providerID.Address,
		HermesID:   hermesID.Address,
	}

	path := "transactor/stake/increase/"
	if waitForBlockchain {
		path += "sync"
	} else {
		path += "async"
	}

	response, err := client.http.Post(path, settleRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not settle promise")
	}
	return nil
}

// SettleWithBeneficiaryStatus set new beneficiary address for the provided identity.
func (client *Client) SettleWithBeneficiaryStatus(address string) (res contract.BeneficiaryTxStatus, err error) {
	response, err := client.http.Get("identities/"+address+"/beneficiary-status", nil)
	if err != nil {
		return contract.BeneficiaryTxStatus{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return contract.BeneficiaryTxStatus{}, fmt.Errorf("expected 200 got %v", response.StatusCode)
	}

	err = parseResponseJSON(response, &res)
	return res, err
}

// SettleWithBeneficiary set new beneficiary address for the provided identity.
func (client *Client) SettleWithBeneficiary(address, beneficiary, hermesID string) error {
	payload := contract.SettleWithBeneficiaryRequest{
		ProviderID:  address,
		HermesID:    hermesID,
		Beneficiary: beneficiary,
	}
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

// DecreaseStake requests the decrease of stake via the transactor.
func (client *Client) DecreaseStake(ID identity.Identity, amount *big.Int) error {
	decreaseRequest := contract.DecreaseStakeRequest{
		ID:     ID.Address,
		Amount: amount,
	}

	path := "transactor/stake/decrease"

	response, err := client.http.Post(path, decreaseRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not decrease stake")
	}
	return nil
}

// WithdrawalHistory returns latest withdrawals for identity
func (client *Client) WithdrawalHistory(address string) (res contract.SettlementListResponse, err error) {
	params := url.Values{
		"types":       []string{"withdrawal"},
		"provider_id": []string{address},
	}

	path := fmt.Sprintf("transactor/settle/history?%s", params.Encode())
	response, err := client.http.Get(path, nil)
	if err != nil {
		return contract.SettlementListResponse{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	return res, err
}

// MigrateHermes migrate from old to active Hermes
func (client *Client) MigrateHermes(address string) error {
	response, err := client.http.Post(fmt.Sprintf("identities/%s/migrate-hermes", address), nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("migration error: %s", body)
	}

	return nil
}

// MigrateHermesStatus check status of the migration
func (client *Client) MigrateHermesStatus(address string) (contract.MigrationStatusResponse, error) {
	var res contract.MigrationStatusResponse

	response, err := client.http.Get(fmt.Sprintf("identities/%s/migrate-hermes/status", address), nil)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)

	return res, err
}

// Beneficiary gets beneficiary address for the provided identity.
func (client *Client) Beneficiary(address string) (res contract.IdentityBeneficiaryResponse, err error) {
	response, err := client.http.Get("identities/"+address+"/beneficiary", nil)
	if err != nil {
		return contract.IdentityBeneficiaryResponse{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	return res, err
}

// SetMMNApiKey sets MMN's API key in config and registers node to MMN
func (client *Client) SetMMNApiKey(data contract.MMNApiKeyRequest) error {
	response, err := client.http.Post("mmn/api-key", data)
	// non 200 status codes return a generic error and we can't use it, instead
	// the response contains validation JSON which we can use to extract the error
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return apierror.Parse(response)
	}

	return nil
}

// IdentityReferralCode returns a referral token for the given identity.
func (client *Client) IdentityReferralCode(identity string) (contract.ReferralTokenResponse, error) {
	response, err := client.http.Get(fmt.Sprintf("identities/%v/referral", identity), nil)
	if err != nil {
		return contract.ReferralTokenResponse{}, err
	}
	defer response.Body.Close()

	res := contract.ReferralTokenResponse{}
	err = parseResponseJSON(response, &res)
	return res, err
}

// OrderCreate creates a new order for currency exchange in pilvytis
func (client *Client) OrderCreate(id identity.Identity, gw string, order contract.PaymentOrderRequest) (contract.PaymentOrderResponse, error) {
	resp, err := client.http.Post(fmt.Sprintf("v2/identities/%s/%s/payment-order", id.Address, gw), order)
	if err != nil {
		return contract.PaymentOrderResponse{}, err
	}
	defer resp.Body.Close()

	var res contract.PaymentOrderResponse
	return res, parseResponseJSON(resp, &res)
}

// OrderGet returns a single order istance given it's ID.
func (client *Client) OrderGet(address identity.Identity, orderID string) (contract.PaymentOrderResponse, error) {
	path := fmt.Sprintf("v2/identities/%s/payment-order/%s", address.Address, orderID)
	resp, err := client.http.Get(path, nil)
	if err != nil {
		return contract.PaymentOrderResponse{}, err
	}
	defer resp.Body.Close()

	var res contract.PaymentOrderResponse
	return res, parseResponseJSON(resp, &res)
}

// OrderGetAll returns all order istances for a given identity
func (client *Client) OrderGetAll(id identity.Identity) ([]contract.PaymentOrderResponse, error) {
	path := fmt.Sprintf("v2/identities/%s/payment-order", id.Address)
	resp, err := client.http.Get(path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res []contract.PaymentOrderResponse
	return res, parseResponseJSON(resp, &res)
}

// OrderInvoice returns a single order istance given it's ID.
func (client *Client) OrderInvoice(address identity.Identity, orderID string) ([]byte, error) {
	path := fmt.Sprintf("v2/identities/%s/payment-order/%s/invoice", address.Address, orderID)
	resp, err := client.http.Get(path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// PaymentOrderGateways returns all possible gateways and their data.
func (client *Client) PaymentOrderGateways(optionsCurrency exchange.Currency) ([]contract.GatewaysResponse, error) {
	query := url.Values{}
	query.Set("options_currency", string(optionsCurrency))
	resp, err := client.http.Get("v2/payment-order-gateways", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res []contract.GatewaysResponse
	return res, parseResponseJSON(resp, &res)
}

// UpdateTerms takes a TermsRequest and sends it as an update
// for the terms of use.
func (client *Client) UpdateTerms(obj contract.TermsRequest) error {
	resp, err := client.http.Post("terms", obj)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// FetchConfig - fetches current config
func (client *Client) FetchConfig() (map[string]interface{}, error) {
	resp, err := client.http.Get("config", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetching config failed with status: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	err = parseResponseJSON(resp, &res)
	if err != nil {
		return nil, err
	}

	data, ok := res["data"]
	if !ok {
		return nil, errors.New("no field named 'data' found in config")
	}

	config := data.(map[string]interface{})
	return config, err
}

// SetConfig - set user config.
func (client *Client) SetConfig(data map[string]interface{}) error {
	req := struct {
		Data map[string]interface{} `json:"data"`
	}{
		Data: data,
	}
	resp, err := client.http.Post("config/user", req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("failed to set user config with status: %d", resp.StatusCode)
	}

	return nil
}
