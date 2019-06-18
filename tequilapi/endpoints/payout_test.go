/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

type mockPayoutInfoRegistry struct {
	recordedID           identity.Identity
	recordedEthAddress   string
	recordedReferralCode string

	mockID           identity.Identity
	mockEthAddress   string
	mockReferralCode string
}

func (mock *mockPayoutInfoRegistry) UpdatePayoutInfo(id identity.Identity, ethAddress string,
	signer identity.Signer) error {
	mock.recordedID = id
	mock.recordedEthAddress = ethAddress
	return nil
}

func (mock *mockPayoutInfoRegistry) UpdateReferralInfo(id identity.Identity, referralCode string,
	signer identity.Signer) error {
	mock.recordedID = id
	mock.recordedReferralCode = referralCode
	return nil
}

func (mock *mockPayoutInfoRegistry) GetPayoutInfo(id identity.Identity, signer identity.Signer) (*mysterium.PayoutInfoResponse, error) {
	if id.Address != mock.mockID.Address {
		return nil, errors.New("payout info for identity is not mocked")
	}

	return &mysterium.PayoutInfoResponse{EthAddress: mock.mockEthAddress, ReferralCode: mock.mockReferralCode}, nil
}

var mockSignerFactory = func(id identity.Identity) identity.Signer { return nil }

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
				"ethAddress": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func TestUpdatePayoutInfo(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		"/irrelevant",
		bytes.NewBufferString(`{"ethAddress": "1234payout", "referral_code": "1234referral"}`),
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

func TestUpdateReferralInfo(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req, err := http.NewRequest(
		http.MethodPut,
		"/irrelevant",
		bytes.NewBufferString(`{"referral_code": "1234referral"}`),
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	mockPayoutInfoRegistry := &mockPayoutInfoRegistry{}
	handlerFunc := NewPayoutEndpoint(mockIdm, mockSignerFactory, mockPayoutInfoRegistry).UpdateReferralInfo
	params := httprouter.Params{{"id", "1234abcd"}}
	handlerFunc(resp, req, params)

	assert.Equal(t, "1234referral", mockPayoutInfoRegistry.recordedReferralCode)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestGetPayoutInfo_ReturnsPayoutInfo(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	mockPayoutInfoRegistry := &mockPayoutInfoRegistry{
		mockID:           existingIdentities[0],
		mockEthAddress:   "mock eth address",
		mockReferralCode: "mock referral code",
	}
	handlerFunc := NewPayoutEndpoint(mockIdm, mockSignerFactory, mockPayoutInfoRegistry).GetPayoutInfo

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.NoError(t, err)
	params := httprouter.Params{{"id", existingIdentities[0].Address}}

	handlerFunc(resp, req, params)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"eth_address": "mock eth address", "referral_code": "mock referral code"}`, resp.Body.String())
}

func TestGetPayoutInfo_ReturnsError_WhenPayoutInfoFindingFails(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	mockPayoutInfoRegistry := &mockPayoutInfoRegistry{mockID: existingIdentities[0], mockEthAddress: "mock eth address"}
	handlerFunc := NewPayoutEndpoint(mockIdm, mockSignerFactory, mockPayoutInfoRegistry).GetPayoutInfo

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.NoError(t, err)
	params := httprouter.Params{{"id", "some other address"}}

	handlerFunc(resp, req, params)

	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.JSONEq(t, `{"message": "payout info for identity is not mocked"}`, resp.Body.String())
}
