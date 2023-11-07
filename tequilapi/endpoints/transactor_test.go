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
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/config"

	"github.com/mysteriumnetwork/node/tequilapi/contract"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/session/pingpong"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
)

var identityRegData = `{
  "beneficiary": "0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
  "fee": 1,
  "stake": 0
}`

func Test_RegisterIdentity(t *testing.T) {
	mockResponse := `{ "fee": 1 }`
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(&registry.FakeRegistry{RegistrationStatus: registry.Unregistered}, tr, a, nil, &settlementHistoryProviderMock{}, &mockAddressProvider{}, nil, nil, &mockPilvytis{})(router)
	assert.NoError(t, err)

	req, err := http.NewRequest(
		http.MethodPost,
		"/identities/0x0000000000000000000000000000000000000000/register",
		bytes.NewBufferString(identityRegData),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func Test_Get_TransactorFees(t *testing.T) {
	mockResponse := `{ "fee": 1000000000000000000 }`
	server := newTestTransactorServer(http.StatusOK, mockResponse)

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, &mockSettler{
		feeToReturn: 11_000,
	}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, nil, nil, nil)(router)
	assert.NoError(t, err)

	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/fees",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t,
		`{
	  "registration": 1000000000000000000,
	  "registration_tokens": {
		"ether": "1",
		"human": "1",
		"wei": "1000000000000000000"
	  },
	  "settlement": 1000000000000000000,
	  "settlement_tokens": {
		"ether": "1",
		"human": "1",
		"wei": "1000000000000000000"
	  },
	  "hermes": 11000,
	  "hermes_percent": "1.1000",
	  "decreaseStake": 1000000000000000000,
	  "decrease_stake_tokens": {
		"ether": "1",
		"human": "1",
		"wei": "1000000000000000000"
	  }
	}
	`,
		resp.Body.String())
}

func Test_SettleAsync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, &mockSettler{}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, &mockBeneficiaryProvider{
		b: common.HexToAddress("0x0000000000000000000000000000000000000001"),
	}, nil, nil)(router)
	assert.NoError(t, err)

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

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, nil, nil, nil)(router)
	assert.NoError(t, err)

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
	assert.Equal(t, "err_hermes_settle_async", apierror.Parse(resp.Result()).Err.Code)
}

func Test_SettleSync_OK(t *testing.T) {
	mockResponse := ""
	server := newTestTransactorServer(http.StatusAccepted, mockResponse)

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, &mockSettler{}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, nil, nil, nil)(router)
	assert.NoError(t, err)

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

	router := summonTestGin()

	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, &mockSettler{errToReturn: errors.New("explosions everywhere")}, &settlementHistoryProviderMock{}, &mockAddressProvider{}, nil, nil, nil)(router)
	assert.NoError(t, err)

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
	assert.Equal(t, "err_hermes_settle", apierror.Parse(resp.Result()).Err.Code)
}

func Test_SettleHistory(t *testing.T) {
	t.Run("returns error on failed history retrieval", func(t *testing.T) {
		mockResponse := ""
		server := newTestTransactorServer(http.StatusAccepted, mockResponse)
		defer server.Close()

		router := summonTestGin()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
		a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
		err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, nil, &settlementHistoryProviderMock{errToReturn: errors.New("explosions everywhere")}, &mockAddressProvider{}, nil, nil, nil)(router)
		assert.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, "/transactor/settle/history", nil)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Equal(t, "err_transactor_settle_history", apierror.Parse(resp.Result()).Err.Code)
	})
	t.Run("returns settlement history", func(t *testing.T) {
		mockStorage := &settlementHistoryProviderMock{settlementHistoryToReturn: []pingpong.SettlementHistoryEntry{
			{
				TxHash:       common.HexToHash("0x88af51047ff2da1e3626722fe239f70c3ddd668f067b2ac8d67b280d2eff39f7"),
				Time:         time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Beneficiary:  common.HexToAddress("0x4443189b9b945DD38E7bfB6167F9909451582eE5"),
				Amount:       big.NewInt(123),
				Fees:         big.NewInt(20),
				IsWithdrawal: true,
			},
			{
				TxHash:       common.HexToHash("0x9eea5c4da8a67929d5dd5d8b6dedb3bd44e7bd3ec299f8972f3212db8afb938a"),
				Time:         time.Date(2020, 6, 7, 8, 9, 10, 0, time.UTC),
				Amount:       big.NewInt(456),
				Fees:         big.NewInt(50),
				IsWithdrawal: true,
			},
		}}

		server := newTestTransactorServer(http.StatusAccepted, "")
		defer server.Close()

		router := summonTestGin()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
		a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
		err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, nil, mockStorage, &mockAddressProvider{}, nil, nil, nil)(router)
		assert.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, "/transactor/settle/history", nil)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(
			t,
			`{
				"items": [
					{
						"tx_hash": "0x88af51047ff2da1e3626722fe239f70c3ddd668f067b2ac8d67b280d2eff39f7",
						"provider_id": "",
						"hermes_id": "0x0000000000000000000000000000000000000000",
						"channel_address": "0x0000000000000000000000000000000000000000",
						"beneficiary":"0x4443189b9B945dD38e7bfB6167F9909451582EE5",
						"amount": 123,
						"settled_at": "2020-01-02T03:04:05Z",
						"fees": 20,
						"is_withdrawal": true,
 						"block_explorer_url": "",
						"error": ""
					},
					{
						"tx_hash": "0x9eea5c4da8a67929d5dd5d8b6dedb3bd44e7bd3ec299f8972f3212db8afb938a",
						"provider_id": "",
						"hermes_id": "0x0000000000000000000000000000000000000000",
						"channel_address": "0x0000000000000000000000000000000000000000",
						"beneficiary": "0x0000000000000000000000000000000000000000",
						"amount": 456,
						"settled_at": "2020-06-07T08:09:10Z",
						"fees": 50,
						"is_withdrawal": true,
 						"block_explorer_url": "",
						"error": ""
					}
				],
				"withdrawal_total": "579",
				"page": 1,
				"page_size": 50,
				"total_items": 2,
				"total_pages": 1
			}`,
			resp.Body.String(),
		)
	})
	t.Run("respects filters", func(t *testing.T) {
		mockStorage := &settlementHistoryProviderMock{}

		server := newTestTransactorServer(http.StatusAccepted, "")
		defer server.Close()
		router := summonTestGin()
		tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil, time.Minute)
		a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
		err := AddRoutesForTransactor(mockIdentityRegistryInstance, tr, a, nil, mockStorage, &mockAddressProvider{}, nil, nil, nil)(router)
		assert.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodGet,
			"/transactor/settle/history?date_from=2020-09-19&date_to=2020-09-20&provider_id=0xab1&hermes_id=0xaB2&types=settlement&types=withdrawal",
			nil,
		)
		assert.Nil(t, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		expectedTimeFrom := time.Date(2020, 9, 19, 0, 0, 0, 0, time.UTC)
		expectedTimeTo := time.Date(2020, 9, 20, 23, 59, 59, 0, time.UTC)
		expectedProviderID := identity.FromAddress("0xab1")
		expectedHermesID := common.HexToAddress("0xaB2")
		expectedTypes := []pingpong.HistoryType{pingpong.SettlementType, pingpong.WithdrawalType}
		assert.Equal(
			t,
			&pingpong.SettlementHistoryFilter{
				TimeFrom:   &expectedTimeFrom,
				TimeTo:     &expectedTimeTo,
				ProviderID: &expectedProviderID,
				HermesID:   &expectedHermesID,
				Types:      expectedTypes,
			},
			mockStorage.calledWithFilter,
		)
	})
}

func Test_AvailableChains(t *testing.T) {
	// given
	router := summonTestGin()
	err := AddRoutesForTransactor(nil, nil, nil, nil, nil, nil, nil, nil, nil)(router)
	assert.NoError(t, err)
	config.Current.SetUser(config.FlagChainID.Name, config.FlagChainID.Value)

	// when
	req, err := http.NewRequest(
		http.MethodGet,
		"/transactor/chain-summary",
		nil,
	)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// then
	var chainSummary contract.ChainSummary
	err = json.NewDecoder(resp.Body).Decode(&chainSummary)
	fmt.Println(chainSummary)
	assert.NoError(t, err)
	assert.Equal(t, "Polygon Mainnet", chainSummary.Chains[137])
	assert.Equal(t, config.FlagChainID.Value, chainSummary.CurrentChain)
}

func Test_Withdrawal(t *testing.T) {
	// given
	router := summonTestGin()

	settler := &mockSettler{
		feeToReturn: 11,
	}
	err := AddRoutesForTransactor(nil, nil, nil, settler, nil, nil, nil, nil, nil)(router)
	assert.NoError(t, err)

	config.Current.SetUser(config.FlagChainID.Name, config.FlagChainID.Value)
	// expect
	for _, data := range []struct {
		fromChainID         int64
		toChainID           int64
		expectedToChainID   int64
		expectedFromChainID int64
	}{
		{fromChainID: 1, toChainID: config.FlagChainID.Value, expectedFromChainID: 1, expectedToChainID: config.FlagChainID.Value},
		{fromChainID: 0, toChainID: 0, expectedFromChainID: config.FlagChainID.Value, expectedToChainID: 0},
		{fromChainID: 1, toChainID: 0, expectedFromChainID: 1, expectedToChainID: 0},
	} {
		t.Run(fmt.Sprintf("succeed withdrawal with fromChainID: %d, toChainID: %d", data.fromChainID, data.toChainID), func(t *testing.T) {
			// when
			body, err := json.Marshal(contract.WithdrawRequest{
				HermesID:    "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				ProviderID:  "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				Beneficiary: "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				ToChainID:   data.toChainID,
				FromChainID: data.fromChainID,
			})
			assert.NoError(t, err)
			req, err := http.NewRequest(
				http.MethodPost,
				"/transactor/settle/withdraw",
				bytes.NewBuffer(body),
			)
			assert.NoError(t, err)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// then
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, data.expectedToChainID, settler.capturedToChainID)
			assert.Equal(t, data.expectedFromChainID, settler.capturedFromChainID)
		})
	}

	// expect
	for _, data := range []struct {
		fromChainID int64
		toChainID   int64
	}{
		{fromChainID: -1, toChainID: 0},
		{fromChainID: 0, toChainID: -1},
		{fromChainID: -1, toChainID: -1},
	} {
		t.Run(fmt.Sprintf("fail withdrawal with unsuported fromChainID: %d, toChainID: %d", data.fromChainID, data.toChainID), func(t *testing.T) {
			// when
			body, err := json.Marshal(contract.WithdrawRequest{
				HermesID:    "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				ProviderID:  "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				Beneficiary: "0xe948dae2ce1faf719ba1091d8c6664a46bab073d",
				ToChainID:   data.toChainID,
				FromChainID: data.fromChainID,
			})
			assert.NoError(t, err)
			req, err := http.NewRequest(
				http.MethodPost,
				"/transactor/settle/withdraw",
				bytes.NewBuffer(body),
			)
			assert.NoError(t, err)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// then
			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	}
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

	capturedToChainID   int64
	capturedFromChainID int64
}

func (ms *mockSettler) ForceSettle(_ int64, _ identity.Identity, _ ...common.Address) error {
	return ms.errToReturn
}

func (ms *mockSettler) ForceSettleAsync(_ int64, _ identity.Identity, _ ...common.Address) error {
	return ms.errToReturn
}

func (ms *mockSettler) SettleIntoStake(_ int64, providerID identity.Identity, hermesID ...common.Address) error {
	return nil
}

func (ms *mockSettler) GetHermesFee(_ int64, _ common.Address) (uint16, error) {
	return ms.feeToReturn, ms.feeErrorToReturn
}

func (ms *mockSettler) Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error {
	ms.capturedToChainID = toChainID
	ms.capturedFromChainID = fromChainID
	return nil
}

type settlementHistoryProviderMock struct {
	settlementHistoryToReturn []pingpong.SettlementHistoryEntry
	errToReturn               error

	calledWithFilter *pingpong.SettlementHistoryFilter
}

func (shpm *settlementHistoryProviderMock) List(filter pingpong.SettlementHistoryFilter) ([]pingpong.SettlementHistoryEntry, error) {
	shpm.calledWithFilter = &filter
	return shpm.settlementHistoryToReturn, shpm.errToReturn
}
