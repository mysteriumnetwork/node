/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mysteriumnetwork/node/config"

	"github.com/stretchr/testify/assert"
)

func Test_EthEndpoints(t *testing.T) {
	// given:
	configFileName := NewTempFileName(t)
	defer os.Remove(configFileName)
	err := config.Current.LoadUserConfig(configFileName)
	assert.NoError(t, err)

	g := summonTestGin()
	err = AddRoutesForValidator(g)
	assert.NoError(t, err)

	// expect:
	for _, data := range []struct {
		payload          string
		expectedResponse int
		configChainId    int64
	}{
		{
			payload:          `["https://invalid"]`,
			configChainId:    80001,
			expectedResponse: 500,
		},
		{
			payload:          `["https://polygon-mumbai.infura.io/v3/e37e62a5c0c44334967779adf83415c4"]`,
			configChainId:    80001,
			expectedResponse: 200,
		},
		{
			payload:          `["https://polygon-mumbai.infura.io/v3/e37e62a5c0c44334967779adf83415c4"]`,
			configChainId:    1,
			expectedResponse: 400,
		},
	} {
		t.Run(fmt.Sprintf("Validate: %s", data.payload), func(t *testing.T) {
			config.Current.SetUser(config.FlagChain2ChainID.Name, data.configChainId)
			req := httptest.NewRequest(
				http.MethodPost,
				"/validation/validate-rpc-chain2-urls",
				strings.NewReader(data.payload))

			resp := httptest.NewRecorder()
			g.ServeHTTP(resp, req)

			assert.Equal(t, data.expectedResponse, resp.Code)
		})
	}
}

func NewTempFileName(t *testing.T) string {
	file, err := os.CreateTemp("", "*")
	assert.NoError(t, err)
	return file.Name()
}
