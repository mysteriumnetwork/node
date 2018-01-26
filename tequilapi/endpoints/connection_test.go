package endpoints

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeManager struct {
	onConnectReturn    error
	onDisconnectReturn error
	onStatusReturn     client_connection.ConnectionStatus
	disconnectCount    int
	requestedIdentity  identity.Identity
	requestedNode      string
}

func (fm *fakeManager) Connect(identity identity.Identity, node string) error {
	fm.requestedIdentity = identity
	fm.requestedNode = node
	return fm.onConnectReturn
}

func (fm *fakeManager) Status() client_connection.ConnectionStatus {

	return fm.onStatusReturn
}

func (fm *fakeManager) Disconnect() error {
	fm.disconnectCount += 1
	return fm.onDisconnectReturn
}

func (fm *fakeManager) Wait() error {
	return nil
}

func TestAddRoutesForConnectionAddsRoutes(t *testing.T) {
	router := httprouter.New()
	fakeManager := fakeManager{}
	ipResolver := ip.NewFakeResolver("123.123.123.123")
	statsStore := &bytescount.SessionStatsStore{}

	AddRoutesForConnection(router, &fakeManager, ipResolver, statsStore)

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
			http.MethodPut, "/connection", `{"identity": "identity", "nodeKey": "nodeKey"}`,
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
			http.StatusOK, `{"bytesSent": 0, "bytesReceived": 0}`,
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
	fakeManager.onStatusReturn = client_connection.ConnectionStatus{
		State:     client_connection.Disconnecting,
		SessionID: "",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
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
	fakeManager.onStatusReturn = client_connection.ConnectionStatus{
		State:     client_connection.NotConnected,
		SessionID: "",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
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
	fakeManager.onStatusReturn = client_connection.ConnectionStatus{
		State: client_connection.Connecting,
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
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
	fakeManager.onStatusReturn = client_connection.ConnectionStatus{
		State:     client_connection.Connected,
		SessionID: "My-super-session",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
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

func TestPutReturns400ErrorIfRequestBodyIsNotJson(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
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

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "validation_error",
			"errors" : {
				"identity" : [ { "code" : "required" , "message" : "Field is required" } ],
				"nodeKey" : [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`, resp.Body.String())
}

func TestPutWithValidBodyCreatesConnection(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"identity" : "my-identity",
				"nodeKey" : "required-node"
			}`))
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)

	assert.Equal(t, identity.FromAddress("my-identity"), fakeManager.requestedIdentity)
	assert.Equal(t, "required-node", fakeManager.requestedNode)

}

func TestDeleteCallsDisconnect(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager, nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	connEndpoint.Kill(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, fakeManager.disconnectCount, 1)
}

func TestGetIPEndpointSucceeds(t *testing.T) {
	manager := fakeManager{}
	ipResolver := ip.NewFakeResolver("123.123.123.123")
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil)
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
	connEndpoint := NewConnectionEndpoint(&manager, ipResolver, nil)
	resp := httptest.NewRecorder()

	connEndpoint.GetIP(resp, nil, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "fake error"
		}`,
		resp.Body.String(),
	)
}

func TestGetStatisticsEndpointReturnsStatistics(t *testing.T) {
	statsStore := &bytescount.SessionStatsStore{}
	stats := bytescount.SessionStats{BytesSent: 1, BytesReceived: 2}
	statsStore.Save(stats)

	manager := fakeManager{}
	connEndpoint := NewConnectionEndpoint(&manager, nil, statsStore)

	resp := httptest.NewRecorder()
	connEndpoint.GetStatistics(resp, nil, nil)
	assert.JSONEq(
		t,
		`{
			"bytesSent": 1,
			"bytesReceived": 2
		}`,
		resp.Body.String(),
	)
}
