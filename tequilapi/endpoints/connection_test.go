/*/*
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

package endpoints

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/payments/crypto"
)

type mockConnectionManager struct {
	onConnectReturn      error
	onDisconnectReturn   error
	onCheckChannelReturn error
	onStatusReturn       connectionstate.Status
	disconnectCount      int
	requestedConsumerID  identity.Identity
	requestedProvider    identity.Identity
	requestedHermesID    common.Address
	requestedServiceType string
}

func (cm *mockConnectionManager) Connect(consumerID identity.Identity, hermesID common.Address, proposalLookup connection.ProposalLookup, options connection.ConnectParams) error {
	proposal, _ := proposalLookup()
	if proposal == nil {
		return errors.New("no proposal")
	}

	cm.requestedConsumerID = consumerID
	cm.requestedHermesID = hermesID
	cm.requestedProvider = identity.FromAddress(proposal.ProviderID)
	cm.requestedServiceType = proposal.ServiceType
	return cm.onConnectReturn
}

func (cm *mockConnectionManager) Status(int) connectionstate.Status {
	return cm.onStatusReturn
}

func (cm *mockConnectionManager) Disconnect(int) error {
	cm.disconnectCount++
	return cm.onDisconnectReturn
}

func (cm *mockConnectionManager) CheckChannel(context.Context) error {
	return cm.onCheckChannelReturn
}

func (cm *mockConnectionManager) Reconnect(int) {
	return
}

func mockRepositoryWithProposal(providerID, serviceType string) *mockProposalRepository {
	sampleProposal := proposal.PricedServiceProposal{
		ServiceProposal: market.ServiceProposal{
			ServiceType: serviceType,
			Location:    TestLocation,
			ProviderID:  providerID,
		},
	}

	return &mockProposalRepository{
		proposals: []proposal.PricedServiceProposal{sampleProposal},
	}
}

func TestAddRoutesForConnectionAddsRoutes(t *testing.T) {
	router := gin.Default()
	state := connectionstate.Status{State: connectionstate.NotConnected}
	fakeManager := &mockConnectionManager{
		onStatusReturn: state,
	}
	fakeState := &mockStateProvider{}
	fakeState.stateToReturn.Connection.Session = state
	fakeState.stateToReturn.Connection.Statistics = connectionstate.Statistics{BytesSent: 1, BytesReceived: 2}

	mockedProposalProvider := mockRepositoryWithProposal("node1", "noop")
	err := AddRoutesForConnection(fakeManager, fakeState, mockedProposalProvider, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(router)
	assert.NoError(t, err)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/connection", "",
			http.StatusOK, `{"status": "NotConnected"}`,
		},
		{
			http.MethodPut, "/connection", `{"consumer_id": "me", "provider_id": "node1", "hermes_id":"hermes", "service_type": "noop"}`,
			http.StatusCreated, `{"status": "NotConnected"}`,
		},
		{
			http.MethodDelete, "/connection", "",
			http.StatusAccepted, "",
		},
		{
			http.MethodGet, "/connection/statistics", "",
			http.StatusOK, `{
				"bytes_sent": 1,
				"bytes_received": 2,
				"throughput_received": 0,
				"throughput_sent": 0,
				"duration": 0,
				"tokens_spent": 0
			}`,
		},
	}

	for _, test := range tests {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		router.ServeHTTP(resp, req)
		assert.Equal(t, test.expectedStatus, resp.Code)
		if test.expectedJSON != "" {
			assert.JSONEq(t, test.expectedJSON, resp.Body.String())
		} else {
			assert.Equal(t, "", resp.Body.String())
		}
	}
}

func TestStateIsReturnedFromStore(t *testing.T) {
	manager := &mockConnectionManager{
		onStatusReturn: connectionstate.Status{
			StartedAt:  time.Time{},
			ConsumerID: identity.Identity{},
			HermesID:   common.Address{},
			State:      connectionstate.Disconnecting,
			SessionID:  "1",
			Proposal:   proposal.PricedServiceProposal{},
		},
	}

	router := gin.Default()
	err := AddRoutesForConnection(manager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(router)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/connection", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "Disconnecting",
			"session_id" : "1"
		}`,
		resp.Body.String(),
	)
}

func TestPutReturns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	fakeManager := mockConnectionManager{}

	router := gin.Default()
	err := AddRoutesForConnection(&fakeManager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(router)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/connection", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "invalid character 'a' looking for beginning of value"
		}`,
		resp.Body.String())
}

func TestPutReturns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	fakeManager := mockConnectionManager{}

	router := gin.Default()
	err := AddRoutesForConnection(&fakeManager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(router)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/connection", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "validation_error",
			"errors" : {
				"consumer_id" : [ { "code" : "required" , "message" : "Field is required" } ]
			}
		}`, resp.Body.String())
}

func TestPutWithValidBodyCreatesConnection(t *testing.T) {
	state := connectionstate.Status{
		State:     connectionstate.Connected,
		SessionID: "1",
	}
	fakeManager := mockConnectionManager{onStatusReturn: state}
	fakeState := &mockStateProvider{}
	fakeState.stateToReturn.Connection.Session = state

	proposalProvider := mockRepositoryWithProposal("required-node", "openvpn")
	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&fakeManager, fakeState, proposalProvider, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, identity.FromAddress("my-identity"), fakeManager.requestedConsumerID)
	assert.Equal(t, common.HexToAddress("hermes"), fakeManager.requestedHermesID)
	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)
	assert.Equal(t, "openvpn", fakeManager.requestedServiceType)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "Connected",
			"session_id" : "1"
		}`,
		resp.Body.String(),
	)
}

func TestPutUnregisteredIdentityReturnsError(t *testing.T) {
	fakeManager := mockConnectionManager{}

	proposalProvider := mockRepositoryWithProposal("required-node", "openvpn")
	mir := *mockIdentityRegistryInstance
	mir.RegistrationStatus = registry.Unregistered

	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&fakeManager, &mockStateProvider{}, proposalProvider, &mir, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusExpectationFailed, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"identity \"my-identity\" is not registered. Please register the identity first"}`,
		resp.Body.String(),
	)
}

func TestPutFailedRegistrationCheckReturnsError(t *testing.T) {
	fakeManager := mockConnectionManager{}

	proposalProvider := mockRepositoryWithProposal("required-node", "openvpn")
	mir := *mockIdentityRegistryInstance
	mir.RegistrationCheckError = errors.New("explosions everywhere")

	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&fakeManager, &mockStateProvider{}, proposalProvider, &mir, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"explosions everywhere"}`,
		resp.Body.String(),
	)
}

func TestPutWithServiceTypeOverridesDefault(t *testing.T) {
	fakeManager := mockConnectionManager{}

	mystAPI := mockRepositoryWithProposal("required-node", "noop")
	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id": "hermes",
				"service_type": "noop"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&fakeManager, &mockStateProvider{}, mystAPI, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)

	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)
	assert.Equal(t, common.HexToAddress("hermes"), fakeManager.requestedHermesID)
	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)
	assert.Equal(t, "noop", fakeManager.requestedServiceType)
}

func TestDeleteCallsDisconnect(t *testing.T) {
	fakeManager := mockConnectionManager{}

	req := httptest.NewRequest(http.MethodDelete, "/connection", nil)
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&fakeManager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, fakeManager.disconnectCount, 1)
}

func TestGetStatisticsEndpointReturnsStatistics(t *testing.T) {
	fakeState := &mockStateProvider{}
	fakeState.stateToReturn.Connection.Statistics = connectionstate.Statistics{BytesSent: 1, BytesReceived: 2}
	fakeState.stateToReturn.Connection.Throughput = bandwidth.Throughput{Up: datasize.BitSpeed(1000), Down: datasize.BitSpeed(2000)}
	fakeState.stateToReturn.Connection.Invoice = crypto.Invoice{AgreementTotal: big.NewInt(10001)}

	manager := mockConnectionManager{}

	resp := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		"/connection/statistics",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id": "hermes",
				"service_type": "noop"
			}`))

	g := gin.Default()
	err := AddRoutesForConnection(&manager, fakeState, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.JSONEq(
		t,
		`{
			"bytes_sent": 1,
			"bytes_received": 2,
			"throughput_sent": 1000,
			"throughput_received": 2000,
			"duration": 0,
			"tokens_spent": 10001
		}`,
		resp.Body.String(),
	)
}

func TestEndpointReturnsConflictStatusIfConnectionAlreadyExists(t *testing.T) {
	manager := mockConnectionManager{}
	manager.onConnectReturn = connection.ErrAlreadyExists

	mystAPI := mockRepositoryWithProposal("required-node", "openvpn")

	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&manager, nil, mystAPI, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "connection already exists"
		}`,
		resp.Body.String(),
	)
}

/*func TestDisconnectReturnsConflictStatusIfConnectionDoesNotExist(t *testing.T) {
	manager := mockConnectionManager{}
	manager.onDisconnectReturn = connection.ErrNoConnection

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})

	req := httptest.NewRequest(
		http.MethodDelete,
		"/irrelevant",
		nil,
	)
	resp := httptest.NewRecorder()

	connectionEndpoint.Kill(&gin.Context{Request: req})

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "no connection exists"
		}`,
		resp.Body.String(),
	)
}*/

func TestConnectReturnsConnectCancelledStatusWhenErrConnectionCancelledIsEncountered(t *testing.T) {
	manager := mockConnectionManager{}
	manager.onConnectReturn = connection.ErrConnectionCancelled

	mockProposalProvider := mockRepositoryWithProposal("required-node", "openvpn")
	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&manager, nil, mockProposalProvider, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, statusConnectCancelled, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "connection was cancelled"
		}`,
		resp.Body.String(),
	)
}

func TestConnectReturnsErrorIfNoProposals(t *testing.T) {
	manager := mockConnectionManager{}
	manager.onConnectReturn = connection.ErrConnectionCancelled

	req := httptest.NewRequest(
		http.MethodPut,
		"/connection",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForConnection(&manager, nil, &mockProposalRepository{}, mockIdentityRegistryInstance, eventbus.New(), &mockAddressProvider{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

var mockIdentityRegistryInstance = &registry.FakeRegistry{RegistrationStatus: registry.Registered}
