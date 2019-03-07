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

type mockPayoutInfoRegistry struct {
	recordedID         identity.Identity
	recordedEthAddress string
}

func (mock *mockPayoutInfoRegistry) UpdatePayoutInfo(id identity.Identity, ethAddress string, signer identity.Signer) error {
	mock.recordedID = id
	mock.recordedEthAddress = ethAddress
	return nil
}

var mockSignerFactory = func(id identity.Identity) identity.Signer { return nil }

/*
func TestUpdatePayoutInfoWithoutAddress(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		"/irrelevant",
		bytes.NewBufferString(`{}`),
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewPayoutEndpoint(mockIdm, mockSignerFactory, nil).UpdatePayoutInfo
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors" : {
				"eth_address": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}
*/
func TestUpdatePayoutInfo(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		"/irrelevant",
		bytes.NewBufferString(`{"ethAddress": "1234payout"}`),
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	mockPayoutInfoRegistry := &mockPayoutInfoRegistry{}
	handlerFunc := NewPayoutEndpoint(mockIdm, mockSignerFactory, mockPayoutInfoRegistry).UpdatePayoutInfo
	params := httprouter.Params{{"id", "1234abcd"}}
	handlerFunc(resp, req, params)

	assert.Equal(t, "1234abcd", mockPayoutInfoRegistry.recordedID.Address)
	assert.Equal(t, "1234payout", mockPayoutInfoRegistry.recordedEthAddress)
	assert.Equal(t, http.StatusOK, resp.Code)
}
