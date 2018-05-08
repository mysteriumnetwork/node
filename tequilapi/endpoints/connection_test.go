package endpoints

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client/connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"github.com/mysterium/node/location"
)

type fakeManager struct {
	onConnectReturn    error
	onDisconnectReturn error
	onStatusReturn     connection.ConnectionStatus
	disconnectCount    int
	requestedConsumer  identity.Identity
	requestedProvider  identity.Identity
}

func (fm *fakeManager) Connect(consumerID identity.Identity, providerID identity.Identity) error {
	fm.requestedConsumer = consumerID
	fm.requestedProvider = providerID
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

func TestAddRoutesForConnectionAddsRoutes(t *testing.T) {
	router := httprouter.New()
	fakeManager := fakeManager{}
	settableClock := utils.SettableClock{}
	ipResolver := ip.NewFakeResolver("123.123.123.123")
	statsKeeper := bytescount.NewSessionStatsKeeper(settableClock.GetTime)
	locationResolver := location.NewResolverFake("current country")
	locationDetector := location.NewDetectorWithLocationResolver(ipResolver, locationResolver)
	originalLocation := location.Location{IP: "100.100.100.100", Country :"original country"}

	sessionStart := time.Date(2000, time.January, 0, 10, 0, 0, 0, time.UTC)
	settableClock.SetTime(sessionStart)
	statsKeeper.MarkSessionStart()
	settableClock.SetTime(sessionStart.Add(time.Minute))



	AddRoutesForConnection(router, &fakeManager, ipResolver, statsKeeper, locationDetector, originalLocation)

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
			http.MethodPut, "/connection", `{"consumerId": "me", "providerId": "node1"}`,
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
		{
			http.MethodGet, "/connection/location", "",
			http.StatusOK,
			`{
				"original": {"ip": "100.100.100.100", "country": "original country"},
				"current":  {"ip": "123.123.123.123", "country": "current country"}
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
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

	assert.Equal(t, identity.FromAddress("my-identity"), fakeManager.requestedConsumer)
	assert.Equal(t, identity.FromAddress("required-node"), fakeManager.requestedProvider)

}

func TestDeleteCallsDisconnect(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil, nil, location.Location{})
	req := httptest.NewRequest(http.MethodDelete, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Kill(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, fakeManager.disconnectCount, 1)
}

func TestGetIPEndpointSucceeds(t *testing.T) {
	manager := fakeManager{}
	ipResolver := ip.NewFakeResolver("123.123.123.123")
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil, nil, location.Location{})
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
	ipResolver := ip.NewFailingFakeResolver(errors.New("fake error"))
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil, nil, location.Location{})
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

func TestGetLocationSucceeds(t *testing.T) {
	manager := fakeManager{}
	ipResolver := ip.NewFakeResolver("123.123.123.123")
	locationResolver := location.NewResolverFake("current country")
	locationDetector := location.NewDetectorWithLocationResolver(ipResolver, locationResolver)
	originalLocation := location.Location{IP: "100.100.100.100", Country :"original country"}
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil, locationDetector, originalLocation)
	resp := httptest.NewRecorder()

	connEndpoint.GetLocation(resp, nil, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"original": {"ip": "100.100.100.100", "country": "original country"},
			"current":  {"ip": "123.123.123.123", "country": "current country"}
		}`,
		resp.Body.String(),
	)
}

func TestGetStatisticsEndpointReturnsStatistics(t *testing.T) {
	settableClock := utils.SettableClock{}
	statsKeeper := bytescount.NewSessionStatsKeeper(settableClock.GetTime)
	stats := bytescount.SessionStats{BytesSent: 1, BytesReceived: 2}
	statsKeeper.Save(stats)

	sessionStart := time.Date(2000, time.January, 0, 10, 0, 0, 0, time.UTC)
	settableClock.SetTime(sessionStart)
	statsKeeper.MarkSessionStart()
	settableClock.SetTime(sessionStart.Add(time.Minute))

	manager := fakeManager{}
	connEndpoint := NewConnectionEndpoint(&manager, nil, statsKeeper, nil, location.Location{})

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
	settableClock := utils.SettableClock{}
	statsKeeper := bytescount.NewSessionStatsKeeper(settableClock.GetTime)
	stats := bytescount.SessionStats{BytesSent: 1, BytesReceived: 2}
	statsKeeper.Save(stats)

	manager := fakeManager{}
	connEndpoint := NewConnectionEndpoint(&manager, nil, statsKeeper, nil, location.Location{})

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

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, nil, location.Location{})

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

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, nil, location.Location{})

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

	connectionEndpoint := NewConnectionEndpoint(&manager, nil, nil, nil, location.Location{})
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
