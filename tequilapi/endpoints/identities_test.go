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
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"

	"github.com/gin-gonic/gin"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

const identityUrl = "/irrelevant"

var (
	existingIdentities = []identity.Identity{
		{Address: "0x000000000000000000000000000000000000000a"},
		{Address: "0x000000000000000000000000000000000000beef"},
	}
	newIdentity = identity.Identity{Address: "0x000000000000000000000000000000000000aaac"}
)

type selectorFake struct {
}

func (hf *selectorFake) UseOrCreate(address, _ string, _ int64) (identity.Identity, error) {
	if len(address) > 0 {
		return identity.Identity{Address: address}, nil
	}

	return identity.Identity{Address: "0x000000"}, nil
}

func (hf *selectorFake) SetDefault(address string) error {
	return nil
}

func TestCurrentIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		"/identities/current",
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{
		idm:      mockIdm,
		selector: &selectorFake{},
	}

	g := gin.Default()
	g.PUT("/identities/current", endpoint.Current)

	g.ServeHTTP(resp, req)

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
		fmt.Sprintf("/identities/%s/unlock", "0x000000000000000000000000000000000000000a"),
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}

	g := gin.Default()
	g.PUT("/identities/:id/unlock", endpoint.Unlock)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, "0x000000000000000000000000000000000000000a", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
	assert.Equal(t, int64(0), mockIdm.LastUnlockChainID)
}

func TestUnlockIdentityWithInvalidJSON(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/identities/%s/unlock", "0x000000000000000000000000000000000000000a"),
		bytes.NewBufferString(`{invalid json}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	g := gin.Default()
	g.PUT("/identities/:id/unlock", endpoint.Unlock)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUnlockIdentityWithNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/identities/%s/unlock", "0x000000000000000000000000000000000000000a"),
		bytes.NewBufferString(`{}`),
	)
	assert.NoError(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	g := gin.Default()
	g.PUT("/identities/:id/unlock", endpoint.Unlock)

	g.ServeHTTP(resp, req)

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
		fmt.Sprintf("/identities/%s/unlock", "0x000000000000000000000000000000000000000a"),
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	assert.Nil(t, err)

	mockIdm.MarkUnlockToFail()

	endpoint := &identitiesAPI{idm: mockIdm}
	g := gin.Default()
	g.PUT("/identities/:id/unlock", endpoint.Unlock)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)

	assert.Equal(t, "0x000000000000000000000000000000000000000a", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
	assert.Equal(t, int64(0), mockIdm.LastUnlockChainID)
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
	g := gin.Default()
	g.POST("/identities", endpoint.Create)

	g.ServeHTTP(resp, req)

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
	g := gin.Default()
	g.POST("/identities", endpoint.Create)

	g.ServeHTTP(resp, req)

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
	g := gin.Default()
	g.POST("/identities", endpoint.Create)

	g.ServeHTTP(resp, req)

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
	path := "/identities"
	req := httptest.NewRequest("GET", path, nil)
	resp := httptest.NewRecorder()

	endpoint := &identitiesAPI{idm: mockIdm}
	g := gin.Default()
	g.GET(path, endpoint.List)

	g.ServeHTTP(resp, req)

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

func Test_ReferralTokenGet(t *testing.T) {
	server := newTestTransactorServer(http.StatusAccepted, `{"token":"yay-free-myst"}`)
	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, &mockAddressProvider{}, fakeSignerFactory, mocks.NewEventBus(), nil)
	endpoint := &identitiesAPI{transactor: tr}

	router := gin.Default()
	router.GET("/identities/:id/referral", endpoint.GetReferralToken)

	tokenRequest := `{"identity": "0x0"}`
	req, err := http.NewRequest(
		http.MethodGet,
		"/identities/0x0/referral",
		bytes.NewBufferString(tokenRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"token":"yay-free-myst"}`, resp.Body.String())
}

func Test_IdentityGet(t *testing.T) {
	endpoint := &identitiesAPI{
		idm:      identity.NewIdentityManagerFake(existingIdentities, newIdentity),
		registry: &registry.FakeRegistry{RegistrationStatus: registry.Registered},
		channelCalculator: &mockAddressProvider{
			channelAddressToReturn: common.HexToAddress("0x100000000000000000000000000000000000000a"),
			hermesToReturn:         common.HexToAddress("0x200000000000000000000000000000000000000a"),
		},
		bc: &mockProviderChannelStatusProvider{
			channelToReturn: client.ProviderChannel{
				Settled:       big.NewInt(1),
				Stake:         big.NewInt(2),
				LastUsedNonce: big.NewInt(3),
				Timelock:      big.NewInt(4),
			},
		},
		earningsProvider: &mockEarningsProvider{
			earnings: pingpongEvent.Earnings{
				LifetimeBalance:  big.NewInt(100),
				UnsettledBalance: big.NewInt(50),
			},
		},
		balanceProvider: &mockBalanceProvider{
			balance: big.NewInt(25),
		},
	}

	router := gin.Default()
	router.GET("/identities/:id", endpoint.Get)

	req, err := http.NewRequest(
		http.MethodGet,
		"/identities/0x000000000000000000000000000000000000000a",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t,
		`
{
  "id": "0x000000000000000000000000000000000000000a",
  "registration_status": "Registered",
  "channel_address": "0x100000000000000000000000000000000000000A",
  "balance": 25,
  "balance_tokens": {
    "wei": "25",
    "ether": "0.000000000000000025",
    "human": "0"
  },
  "earnings": 50,
  "earnings_total": 100,
  "stake": 2,
  "hermes_id": "0x200000000000000000000000000000000000000A"
}
`,
		resp.Body.String())
}

type mockAddressProvider struct {
	hermesToReturn         common.Address
	registryToReturn       common.Address
	channelToReturn        common.Address
	channelAddressToReturn common.Address
}

func (ma *mockAddressProvider) GetChannelImplementation(chainID int64) (common.Address, error) {
	return ma.channelToReturn, nil
}

func (ma *mockAddressProvider) GetActiveHermes(chainID int64) (common.Address, error) {
	return ma.hermesToReturn, nil
}

func (ma *mockAddressProvider) GetRegistryAddress(chainID int64) (common.Address, error) {
	return ma.registryToReturn, nil
}

func (ma *mockAddressProvider) GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error) {
	return ma.channelAddressToReturn, nil
}

type mockProviderChannelStatusProvider struct {
	channelToReturn client.ProviderChannel
}

func (m *mockProviderChannelStatusProvider) GetProviderChannel(chainID int64, hermesAddress common.Address, provider common.Address, pending bool) (client.ProviderChannel, error) {
	return m.channelToReturn, nil
}

type mockEarningsProvider struct {
	earnings pingpongEvent.Earnings
	channels []pingpong.HermesChannel
}

func (mep *mockEarningsProvider) List(chainID int64) []pingpong.HermesChannel {
	return mep.channels
}

func (mep *mockEarningsProvider) GetEarnings(chainID int64, _ identity.Identity) pingpongEvent.Earnings {
	return mep.earnings
}

type mockBalanceProvider struct {
	balance            *big.Int
	forceUpdateBalance *big.Int
}

func (m *mockBalanceProvider) GetBalance(chainID int64, id identity.Identity) *big.Int {
	return m.balance
}
func (m *mockBalanceProvider) ForceBalanceUpdateCached(chainID int64, id identity.Identity) *big.Int {
	return m.forceUpdateBalance
}
