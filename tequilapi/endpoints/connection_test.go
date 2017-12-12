package endpoints

import (
	"bytes"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeManager struct {
	onConnectReturn    error
	onDisconnectReturn error
	onStatusReturn     client_connection.Status
	disconnectCount    int
	requestedIdentity  dto.Identity
	requestedNode      string
}

func (fm *fakeManager) Connect(identity dto.Identity, node string) error {
	fm.requestedIdentity = identity
	fm.requestedNode = node
	return fm.onConnectReturn
}

func (fm *fakeManager) Status() client_connection.Status {

	return fm.onStatusReturn
}

func (fm *fakeManager) Disconnect() error {
	fm.disconnectCount += 1
	return fm.onDisconnectReturn
}

func (fm *fakeManager) Wait() error {
	return nil
}

func TestNotConnectedStateIsReturnedWhenNoConnection(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = client_connection.Status{
		State:     client_connection.NOT_CONNECTED,
		SessionId: "",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(http.MethodGet, "/connection", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "NOT_CONNECTED"
		}`,
		resp.Body.String())
}

func TestConnectedStateAndSessionIdIsReturnedWhenIsConnection(t *testing.T) {
	var fakeManager = fakeManager{}
	fakeManager.onStatusReturn = client_connection.Status{
		State:     client_connection.CONNECTED,
		SessionId: "My-super-session",
	}

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(http.MethodGet, "/connection", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"status" : "CONNECTED",
			"sessionId" : "My-super-session"
		}`,
		resp.Body.String())

}

func TestPutReturns400ErrorIfRequestBodyIsNotJson(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(http.MethodPut, "/connection", bytes.NewBufferString("a"))
	assert.Nil(t, err)
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

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(http.MethodPut, "/connection", bytes.NewBufferString("{}"))
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	assert.JSONEq(
		t,
		`{
			"message" : "validation_error",
			"validationErros" : [
				{ "field" : "" , "error" : ""} ,
				{ "field" : "" , "error" : ""}
			]
		}`,
		resp.Body.String())

}

func TestPutWithValidBodyCreatesConnection(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(
		http.MethodPut,
		"/connection",
		bytes.NewBufferString(
			`{
				"identity" : "my-identity",
				"nodeKey" : "required-node"
			}`))
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Create(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)

	assert.Equal(t, dto.Identity("my-identity"), fakeManager.requestedIdentity)
	assert.Equal(t, "required-node", fakeManager.requestedNode)

}

func TestDeleteCallsDisconnect(t *testing.T) {
	fakeManager := fakeManager{}

	connEndpoint := NewConnectionEndpoint(&fakeManager)
	req, err := http.NewRequest(http.MethodDelete, "/connection", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	connEndpoint.Kill(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, fakeManager.disconnectCount, 1)
}
