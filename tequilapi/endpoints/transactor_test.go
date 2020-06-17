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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/session/pingpong"
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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{})

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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "registryAddress", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "hermesID", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, &mockSettler{
		feeToReturn: 11,
	}, &settlementHistoryProviderMock{})

	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/fees",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"registration":1, "settlement":1, "hermes":11}`, resp.Body.String())
}

func Test_TopUp_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := httprouter.New()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{})

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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0x599d43715DF3070f83355D9D90AE62c159E62A75", "0x599d43715DF3070f83355D9D90AE62c159E62A75", "0x599d43715DF3070f83355D9D90AE62c159E62A75", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{})

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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, &mockSettler{}, &settlementHistoryProviderMock{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{})

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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, &mockSettler{}, &settlementHistoryProviderMock{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
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

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	AddRoutesForTransactor(router, tr, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{})

	settleRequest := `{"hermes_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "provider_id": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10"}`
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

func Test_SettleHistory(t *testing.T) {
	t.Run("returns error on missing providerID", func(t *testing.T) {
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{})

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"message":"providerID is required"}`, resp.Body.String())
	})
	t.Run("returns error on missing hermesID", func(t *testing.T) {
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{})

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history?providerID=0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"message":"hermesID is required"}`, resp.Body.String())
	})
	t.Run("returns error on failed history retrieval", func(t *testing.T) {
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{errToReturn: errors.New("explosions everywhere")})

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history?providerID=0xbe180c8CA53F280C7BE8669596fF7939d933AA10&hermesID=0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"message":"explosions everywhere"}`, resp.Body.String())
	})
	t.Run("returns settlement history", func(t *testing.T) {
		expectedJSON := `[{"time":"2020-05-26T09:12:32.475904Z","tx_hash":"0x88af51047ff2da1e3626722fe239f70c3ddd668f067b2ac8d67b280d2eff39f7","promise":{"ChannelID":"+6pGXXkM8mIuwjZ72JNukxcqE1UkhbC47Ijg4UqurJY=","Amount":30245,"Fee":0,"Hashlock":"6RauIa1dz9pXOca788BIJigdgIzWVbn9k3VXZLvp2gM=","R":"qhLbkT/xKQFxvf7bE66yA/mr4LY5WEnqP/280KnNHi8=","Signature":"pqHSWofpfM7cUZ2KfhKmzd+iyxs6xsbWqXOkO0noCRwEfHHZtAP2S9E+sE72m2bFQmTtPB8mzQ6X0aNrn9h39Rw="},"beneficiary":"0x0000000000000000000000000000000000000000","amount":30091,"total_settled":30245},{"time":"2020-05-26T08:15:58.386698Z","tx_hash":"0x9eea5c4da8a67929d5dd5d8b6dedb3bd44e7bd3ec299f8972f3212db8afb938a","promise":{"ChannelID":"+6pGXXkM8mIuwjZ72JNukxcqE1UkhbC47Ijg4UqurJY=","Amount":154,"Fee":0,"Hashlock":"wIAKURZIMqlrlyjXNOX+Y8xIDyPSBZuMFU37bNJrRNQ=","R":"PqnMB6sYiwgM+pPdnk5Q8TOw93E78M7aFz1G2DkBKJE=","Signature":"elsD3ennGajAjUg7Ky4M4+8Olde2V2vNwm2v1c5pqQs/6V0mY7ECPLUzsU8dGKKI5EceFUVGqTnKrcLIUwRY6xs="},"beneficiary":"0x0000000000000000000000000000000000000000","amount":154,"total_settled":154}]`
		res := []pingpong.SettlementHistoryEntry{}
		err := json.Unmarshal([]byte(expectedJSON), &res)
		assert.Nil(t, err)
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := httprouter.New()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
		AddRoutesForTransactor(router, tr, nil, &settlementHistoryProviderMock{settlementHistoryToReturn: res})

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history?providerID=0xbe180c8CA53F280C7BE8669596fF7939d933AA10&hermesID=0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, expectedJSON, resp.Body.String())
	})
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

	feeToReturn      uint16
	feeErrorToReturn error
}

func (ms *mockSettler) ForceSettle(_ identity.Identity, _ common.Address) error {
	return ms.errToReturn
}

func (ms *mockSettler) SettleWithBeneficiary(_ identity.Identity, _, _ common.Address) error {
	return ms.errToReturn
}

func (ms *mockSettler) GetHermesFee() (uint16, error) {
	return ms.feeToReturn, ms.feeErrorToReturn
}

type settlementHistoryProviderMock struct {
	settlementHistoryToReturn []pingpong.SettlementHistoryEntry
	errToReturn               error
}

func (shpm *settlementHistoryProviderMock) Get(provider identity.Identity, hermes common.Address) ([]pingpong.SettlementHistoryEntry, error) {
	return shpm.settlementHistoryToReturn, shpm.errToReturn
}
