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
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
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

func TestUnlockIdentityWithNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{}`),
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewIdentitiesEndpoint(mockIdm, mystClient, fakeSignerFactory).Unlock
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
