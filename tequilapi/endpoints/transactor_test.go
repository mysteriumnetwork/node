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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/transactor"
	"github.com/mysteriumnetwork/node/identity"
)

var identityRegData = `{
  "beneficiary": "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c",
  "fee": 1,
  "stake": 0
}`

func Test_RegisterIdentity(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := transactor.NewTransactor(server.URL, server.URL, "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", fakeSignerFactory)
	AddRoutesForTransactor(router, tr)

	req, err := http.NewRequest(
		http.MethodPost,
		"/identities/{id}/register",
		bytes.NewBufferString(identityRegData),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_Get_TransactorFees(t *testing.T) {
	mockResponse := `{ "transaction": 1, "registration": 100 }`
	server := newTestTransactorServer(http.StatusOK, mockResponse)

	router := httprouter.New()

	tr := transactor.NewTransactor(server.URL, server.URL, "registryAddress", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "accountantID", fakeSignerFactory)
	AddRoutesForTransactor(router, tr)

	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/fees",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, mockResponse, resp.Body.String())
}

func Test_TopUp_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := transactor.NewTransactor(server.URL, server.URL, "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", fakeSignerFactory)
	AddRoutesForTransactor(router, tr)

	topUpData := `{"identity": "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/topup",
		bytes.NewBufferString(topUpData),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_TopUp_BubblesErrors(t *testing.T) {
	mockResponse := ""
	mockStatus := http.StatusBadGateway
	server := newTestTransactorServer(mockStatus, mockResponse)

	router := httprouter.New()

	tr := transactor.NewTransactor(server.URL, server.URL, "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c", fakeSignerFactory)
	AddRoutesForTransactor(router, tr)

	topUpData := `{"identity": "0x241F6e1d0bB17f45767DC60A6Bd3D21Cdb543a0c"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/topup",
		bytes.NewBufferString(topUpData),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(
		t,
		fmt.Sprintf(`{"message":"server response invalid: %v %v (%v/fee/topup)"}`, mockStatus, http.StatusText(mockStatus), server.URL),
		resp.Body.String(),
	)
}

func newTestTransactorServer(mockStatus int, mockResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResponse))
	}))
}

var fakeSignerFactory = func(id identity.Identity) identity.Signer {
	return &fakeSigner{}
}

type fakeSigner struct {
}

func pad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	tmp := make([]byte, size)
	copy(tmp[size-len(b):], b)
	return tmp
}

func (fs *fakeSigner) Sign(message []byte) (identity.Signature, error) {
	b := make([]byte, 65)
	b = pad(b, 65)
	return identity.SignatureBytes(b), nil
}
