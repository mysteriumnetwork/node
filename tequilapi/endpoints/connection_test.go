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

package endpoints

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

type fakeManager struct {
	onConnectReturn      error
	onDisconnectReturn   error
	onStatusReturn       connection.ConnectionStatus
	disconnectCount      int
	requestedConsumerID  identity.Identity
	requestedProvider    identity.Identity
	requestedServiceType string
}

func (fm *fakeManager) Connect(consumerID identity.Identity, proposal market.ServiceProposal, options connection.ConnectParams) error {
	fm.requestedConsumerID = consumerID
	fm.requestedProvider = identity.FromAddress(proposal.ProviderID)
	fm.requestedServiceType = proposal.ServiceType
	return fm.onConnectReturn
}

func (fm *fakeManager) Status() connection.ConnectionStatus {

	return fm.onStatusReturn
}

func (fm *fakeManager) Disconnect() error {
	fm.disconnectCount++
	return fm.onDisconnectReturn
}

func (fm *fakeManager) Wait() error {
	return nil
}

type StubStatisticsTracker struct {
	duration time.Duration
	stats    consumer.SessionStatistics
}

func (ssk *StubStatisticsTracker) Retrieve() consumer.SessionStatistics {
	return ssk.stats
}

func (ssk *StubStatisticsTracker) GetSessionDuration() time.Duration {
	return ssk.duration
}

func getMockProposalProviderWithSpecifiedProposal(providerID, serviceType string) ProposalProvider {
	sampleProposal := market.ServiceProposal{
		ID:                1,
		ServiceType:       serviceType,
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        providerID,
	}

	return &mockProposalProvider{
		proposals: []market.ServiceProposal{sampleProposal},
	}
}

func TestAddRoutesForConnectionAddsRoutes(t *testing.T) {
	router := httprouter.New()
	fakeManager := fakeManager{}
	statsKeeper := &StubStatisticsTracker{
		duration: time.Minute,
	}
	ipResolver := ip.NewResolverFake("123.123.123.123")

	mockedProposalProvider := getMockProposalProviderWithSpecifiedProposal("node1", "noop")
	AddRoutesForConnection(router, &fakeManager, ipResolver, statsKeeper, mockedProposalProvider)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/connection", "",
			http.StatusOK, `{"status": ""}`,
		},
		{
			http.MethodPut, "/connection", `{"consumerId": "me", "providerId": "node1", "serviceType": "noop"}`,
			http.StatusCreated, `{"status": ""}`,
		},
		{
			http.MethodDelete, "/connection", "",
			http.StatusAccepted, "",
		},
		{
			http.MethodGet, "/connection/ip", "",
			http.StatusOK, `{"ip": "123.123.123.123"}`,
		},
		{
			http.MethodGet, "/connection/statistics", "",
			http.StatusOK, `{
				"bytesSent": 0,
				"bytesReceived": 0,
				"duration": 60
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

func TestDisconnectingState(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State:     connection.Disconnecting,
		SessionID: "",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "Disconnecting"
		}`,
		resp.Body.String())
}

func TestNotConnectedStateIsReturnedWhenNoConnection(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State:     connection.NotConnected,
		SessionID: "",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
            "status" : "NotConnected"
        }`,
		resp.Body.String(),
	)
}

func TestStateConnectingIsReturnedWhenIsConnectionInProgress(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State: connection.Connecting,
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
            "status" : "Connecting"
        }`,
		resp.Body.String(),
	)
}

func TestConnectedStateAndSessionIdIsReturnedWhenIsConnected(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = connection.ConnectionStatus{
		State:     connection.Connected,
		SessionID: "My-super-session",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "Connected",
			"sessionId" : "My-super-session"
		}`,
		resp.Body.String())

}

func TestPutReturns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "invalid character 'a' looking for beginning of value"
		}`,
		resp.Body.String())
}

func TestPutReturns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "validation_error",
			"errors" : {
				"consumerId" : [ { "code" : "required" , "message" : "Field is required" } ],
				"providerId" : [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`, resp.Body.String())
}

func TestPutWithValidBodyCreatesConnection(t *testing.T) {
	fakeManager := fakeManager{}

	proposalProvider := getMockProposalProviderWithSpecifiedProposal("required-node", "openvpn")
	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, proposalProvider)
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumerId" : "my-identity",
				"providerId" : "required-node"
			}`))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)

	assert.Equal(t, identity.FromAddress("my-identity"), fakeManager.requestedConsumerID)
	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)
	assert.Equal(t, "openvpn", fakeManager.requestedServiceType)
}

func TestPutWithServiceTypeOverridesDefault(t *testing.T) {
	fakeManager := fakeManager{}

	mystAPI := getMockProposalProviderWithSpecifiedProposal("required-node", "noop")
	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, mystAPI)
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumerId" : "my-identity",
				"providerId" : "required-node",
				"serviceType": "noop"
			}`))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)

	assert.Equal(t, identity.FromAddress("my-identity"), fakeManager.requestedConsumerID)
	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)
	assert.Equal(t, "noop", fakeManager.requestedServiceType)
}

func TestDeleteCallsDisconnect(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, &mockProposalProvider{})
	req := httptest.NewRequest(http.MethodDelete, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Kill(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, fakeManager.disconnectCount, 1)
}

func TestGetIPEndpointSucceeds(t *testing.T) {
	manager := fakeManager{}
	ipResolver := ip.NewResolverFake("123.123.123.123")
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil, &mockProposalProvider{})
	resp := httptest.NewRecorder()

	connEndpoint.GetIP(resp, nil, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"ip": "123.123.123.123"
		}`,
		resp.Body.String(),
	)
}

func TestGetIPEndpointReturnsErrorWhenIPDetectionFails(t *testing.T) {
	manager := fakeManager{}
	ipResolver := ip.NewResolverFakeFailing(errors.New("fake error"))
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil, &mockProposalProvider{})
	resp := httptest.NewRecorder()

	connEndpoint.GetIP(resp, nil, nil)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "fake error"
		}`,
		resp.Body.String(),
	)
}

func TestGetStatisticsEndpointReturnsStatistics(t *testing.T) {
	statsKeeper := &StubStatisticsTracker{
		duration: time.Minute,
		stats:    consumer.SessionStatistics{BytesSent: 1, BytesReceived: 2},
	}

	manager := fakeManager{}
	connEndpoint := NewConnectionEndpoint(&manager, nil, statsKeeper, &mockProposalProvider{})

	resp := httptest.NewRecorder()
	connEndpoint.GetStatistics(resp, nil, nil)
	assert.JSONEq(
		t,
		`{
			"bytesSent": 1,
			"bytesReceived": 2,
			"duration": 60
		}`,
		resp.Body.String(),
	)
}

func TestGetStatisticsEndpointReturnsStatisticsWhenSessionIsNotStarted(t *testing.T) {
	statsKeeper := &StubStatisticsTracker{
		stats: consumer.SessionStatistics{BytesSent: 1, BytesReceived: 2},
	}

	manager := fakeManager{}
	connEndpoint := NewConnectionEndpoint(&manager, nil, statsKeeper, &mockProposalProvider{})

	resp := httptest.NewRecorder()
	connEndpoint.GetStatistics(resp, nil, nil)
	assert.JSONEq(
		t,
		`{
			"bytesSent": 1,
			"bytesReceived": 2,
			"duration": 0
		}`,
		resp.Body.String(),
	)
}

func TestEndpointReturnsConflictStatusIfConnectionAlreadyExists(t *testing.T) {
	manager := fakeManager{}
	manager.onConnectReturn = connection.ErrAlreadyExists

	mystAPI := getMockProposalProviderWithSpecifiedProposal("required-node", "openvpn")
	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, mystAPI)

	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumerId" : "my-identity",
				"providerId" : "required-node"
			}`))
	resp := httptest.NewRecorder()

	connectionEndpoint.Create(resp, req, nil)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "connection already exists"
		}`,
		resp.Body.String(),
	)
}

func TestDisconnectReturnsConflictStatusIfConnectionDoesNotExist(t *testing.T) {
	manager := fakeManager{}
	manager.onDisconnectReturn = connection.ErrNoConnection

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, &mockProposalProvider{})

	req := httptest.NewRequest(
		http.MethodDelete,
		"/irrelevant",
		nil,
	)
	resp := httptest.NewRecorder()

	connectionEndpoint.Kill(resp, req, nil)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "no connection exists"
		}`,
		resp.Body.String(),
	)
}

func TestConnectReturnsConnectCancelledStatusWhenErrConnectionCancelledIsEncountered(t *testing.T) {
	manager := fakeManager{}
	manager.onConnectReturn = connection.ErrConnectionCancelled

	mockProposalProvider := getMockProposalProviderWithSpecifiedProposal("required-node", "openvpn")
	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, mockProposalProvider)
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumerId" : "my-identity",
				"providerId" : "required-node"
			}`))
	resp := httptest.NewRecorder()

	connectionEndpoint.Create(resp, req, nil)

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
	manager := fakeManager{}
	manager.onConnectReturn = connection.ErrConnectionCancelled

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, &mockProposalProvider{proposals: make([]market.ServiceProposal, 0)})
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumerId" : "my-identity",
				"providerId" : "required-node"
			}`))
	resp := httptest.NewRecorder()

	connectionEndpoint.Create(resp, req, nil)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "provider has no service proposals"
		}`,
		resp.Body.String(),
	)
}
