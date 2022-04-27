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

package observer

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestConsumerObserverAPI(t *testing.T) {
	defaultResp := map[int64][]HermesResponse{
		1: {HermesResponse{
			HermesAddress:      common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Operator:           common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Version:            0,
			Approved:           true,
			ChannelImplAddress: common.HexToAddress("0x0000000000000000000000000000000000000003"),
			Fee:                100,
			URL:                "something.com",
		}},
		137: {HermesResponse{
			HermesAddress:      common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Operator:           common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Version:            0,
			Approved:           true,
			ChannelImplAddress: common.HexToAddress("0x0000000000000000000000000000000000000003"),
			Fee:                100,
			URL:                "something.com",
		},
			HermesResponse{
				HermesAddress:      common.HexToAddress("0x0000000000000000000000000000000000000006"),
				Operator:           common.HexToAddress("0x0000000000000000000000000000000000000007"),
				Version:            1,
				Approved:           true,
				ChannelImplAddress: common.HexToAddress("0x0000000000000000000000000000000000000008"),
				Fee:                400,
				URL:                "something2.com",
			}},
	}
	httpClient := &mockHttpClient{
		response:    defaultResp,
		timesCalled: 0,
	}
	api := NewAPI(httpClient, "url")
	t.Run("uses httpclient if not present", func(t *testing.T) {
		data, err := api.GetHermesesData()
		assert.NoError(t, err)
		assert.Len(t, data, 2)
		assert.Len(t, data[137], 2)

		assert.Equal(t, 1, httpClient.getTimesCalled())
	})

	t.Run("uses cached value if present and valid", func(t *testing.T) {
		data, err := api.GetHermesesData()
		assert.NoError(t, err)
		assert.Len(t, data, 2)
		assert.Len(t, data[137], 2)

		assert.Equal(t, 1, httpClient.getTimesCalled())
	})

	t.Run("updates value if present and not valid", func(t *testing.T) {
		httpClient.response = map[int64][]HermesResponse{
			137: {HermesResponse{
				HermesAddress:      common.HexToAddress("0x0000000000000000000000000000000000000001"),
				Operator:           common.HexToAddress("0x0000000000000000000000000000000000000002"),
				Version:            0,
				Approved:           true,
				ChannelImplAddress: common.HexToAddress("0x0000000000000000000000000000000000000003"),
				Fee:                100,
				URL:                "something.com",
			},
			},
		}
		api.cachedResponse.ValidUntil = time.Now().Add(-time.Hour)
		data, err := api.GetHermesesData()
		assert.NoError(t, err)
		assert.Len(t, data, 1)
		assert.Len(t, data[137], 1)

		assert.Equal(t, 2, httpClient.getTimesCalled())
	})
}

type mockHttpClient struct {
	response    HermesesResponse
	timesCalled int
}

func (mhc *mockHttpClient) DoRequestAndParseResponse(req *http.Request, resp interface{}) error {
	mhc.timesCalled++
	data, err := json.Marshal(mhc.response)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, resp)
}

func (mhc *mockHttpClient) getTimesCalled() int {
	return mhc.timesCalled
}
