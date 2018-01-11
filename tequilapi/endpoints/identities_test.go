package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/stretchr/testify/assert"
)

const identityUrl = "/irrelevant"

var (
	mockIdm = identity.NewIdentityManagerFake([]identity.Identity{
		{"0x000000000000000000000000000000000000000a"},
		{"0x000000000000000000000000000000000000beef"},
	}, identity.Identity{"0x000000000000000000000000000000000000aaac"})
	mystClient        = server.NewClientFake()
	fakeSignerFactory = func(id identity.Identity) identity.Signer { return nil } //it works in this case - it's passed to fake myst client
)

func TestRegisterExistingIdentityRequest(t *testing.T) {
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

func TestCreateNewIdentityEmptyPassword(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"password": ""}`),
	)

	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Create
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestCreateNewIdentityNoPassword(t *testing.T) {
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
				"password": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func TestCreateNewIdentity(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"password": "mypass"}`),
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
