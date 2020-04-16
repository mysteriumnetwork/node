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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
)

var identityRegData = `{
  "beneficiary": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
  "fee": 1,
  "stake": 0
}`

func Test_RegisterIdentity(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, nil)

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
	mockResponse := `{ "fee": 1 }`
	server := newTestTransactorServer(http.StatusOK, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "registryAddress", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "accountantID", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, nil)

	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/fees",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"registration":1, "settlement":1}`, resp.Body.String())
}

func Test_TopUp_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, nil)

	topUpData := `{"identity": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0x599d43715DF3070f83355D9D90AE62c159E62A75", "0x599d43715DF3070f83355D9D90AE62c159E62A75", "0x599d43715DF3070f83355D9D90AE62c159E62A75", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, nil)

	topUpData := `{"identity": "0x599d43715DF3070f83355D9D90AE62c159E62A75"}`
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
		fmt.Sprintf(`{"message":"server response invalid: %v %v (%v/topup)"}`, mockStatus, http.StatusText(mockStatus), server.URL),
		resp.Body.String(),
	)
}

func Test_SettleAsync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, &mockSettler{})

	settleRequest := `{"accountant_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/async",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_SettleAsync_ReturnsError(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")})

	settleRequest := `asdasdasd`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/async",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"message":"failed to unmarshal settle request: invalid character 'a' looking for beginning of value"}`, resp.Body.String())
}

func Test_SettleSync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, &mockSettler{})

	settleRequest := `{"accountant_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/sync",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_SettleSync_ReturnsError(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus())
	AddRoutesForTransactor(router, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")})

	settleRequest := `{"accountant_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
	req, err := http.NewRequest(
		http.MethodPost,
		"/transactor/settle/sync",
		bytes.NewBufferString(settleRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"message":"settling failed: explosions everywhere"}`, resp.Body.String())
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

type mockSettler struct {
	errToReturn error
}

func (ms *mockSettler) ForceSettle(_ identity.Identity, _ common.Address) error {
	return ms.errToReturn
}
