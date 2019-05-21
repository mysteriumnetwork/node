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
	"github.com/stretchr/testify/assert"
)

const identityUrl = "/irrelevant"

var (
	existingIdentities = []identity.Identity{
		{"0x000000000000000000000000000000000000000a"},
		{"0x000000000000000000000000000000000000beef"},
	}
	newIdentity = identity.Identity{"0x000000000000000000000000000000000000aaac"}
)

type selectorFake struct {
	id *identity.Identity
}

func (hf *selectorFake) setIdentity(id *identity.Identity) {
	hf.id = id
}
func (hf *selectorFake) UseOrCreate(address, passphrase string) (identity.Identity, error) {
	if len(address) > 0 {
		return identity.Identity{Address: address}, nil
	}

	return identity.Identity{Address: "0x000000"}, nil
}

func TestCurrentIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{"id", "current"}}
	assert.Nil(t, err)

	endpoint := &identitiesAPI{
		idm:      mockIdm,
		selector: &selectorFake{},
	}
	endpoint.Current(resp, req, params)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id": "0x000000"
		}`,
		resp.Body.String(),
	)
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

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

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

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUnlockIdentityWithNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{}`),
	)
	assert.NoError(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, nil)

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

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusForbidden, resp.Code)

	assert.Equal(t, "1234abcd", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
}

func TestCreateNewIdentityEmptyPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": ""}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestCreateNewIdentityNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

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
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": "mypass"}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

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

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.List(resp, req, nil)

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
