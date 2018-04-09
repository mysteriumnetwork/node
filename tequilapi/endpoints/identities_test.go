package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/stretchr/testify/assert"
)

const identityUrl = "/irrelevant"

var (
	existingIdentities = []identity.Identity{
		{"0x000000000000000000000000000000000000000a"},
		{"0x000000000000000000000000000000000000beef"},
	}
	newIdentity       = identity.Identity{"0x000000000000000000000000000000000000aaac"}
	mystClient        = server.NewClientFake()
	fakeSignerFactory = func(id identity.Identity) identity.Signer { return nil } //it works in this case - it's passed to fake myst client
)

func TestRegisterExistingIdentityRequest(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"registered": false}`),
	)

	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Register
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusNotImplemented, resp.Code)
	assert.JSONEq(
		t,
		`{"message": "Unregister not supported"}`,
		resp.Body.String(),
	)
}

func TestRegisterIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"registered": true}`),
	)

	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Register
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusAccepted, resp.Code)
}

func TestUnlockIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{"id", "1234abcd"}}
	assert.Nil(t, err)

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Unlock
	handlerFunc(resp, req, params)

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, "1234abcd", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
}

func TestUnlockIdentityWithInvalidJSON(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{invalid json}`),
	)
	params := httprouter.Params{{"id", "1234abcd"}}
	assert.Nil(t, err)

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Unlock
	handlerFunc(resp, req, params)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUnlockFailure(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{"id", "1234abcd"}}
	assert.Nil(t, err)

	mockIdm.MarkUnlockToFail()

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Unlock
	handlerFunc(resp, req, params)

	assert.Equal(t, http.StatusForbidden, resp.Code)

	assert.Equal(t, "1234abcd", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
}

func TestCreateNewIdentityEmptyPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": ""}`),
	)

	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Create
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestCreateNewIdentityNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{}`),
	)

	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Create
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors" : {
				"passphrase": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func TestCreateNewIdentity(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": "mypass"}`),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Create
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "id": "0x000000000000000000000000000000000000aaac"
        }`,
		resp.Body.String(),
	)
}

func TestListIdentities(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "identities": [
                {"id": "0x000000000000000000000000000000000000000a"},
                {"id": "0x000000000000000000000000000000000000beef"}
            ]
        }`,
		resp.Body.String(),
	)
}
