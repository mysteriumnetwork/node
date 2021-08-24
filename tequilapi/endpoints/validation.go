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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type validationEndpoints struct {
}

// ValidateRPCChain2URLS validates list of RPC Chain2 urls
// swagger:operation GET /validation/validate-rpc-chain2-urls
// ---
// summary: validates list of RPC Chain2 urls
// description: validates list of RPC Chain2 urls
// response:
//	200:
//    description: Validation success
//  400:
//	  description: validation failed
func (e validationEndpoints) ValidateRPCChain2URLS(c *gin.Context) {
	req := c.Request
	resp := c.Writer

	var rpcURLS []string
	err := json.NewDecoder(req.Body).Decode(&rpcURLS)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	for _, rpc := range rpcURLS {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client, err := ethclient.DialContext(ctx, rpc)
		if err != nil {
			utils.SendError(resp, err, http.StatusInternalServerError)
			return
		}

		rpcURLChainID, err := client.ChainID(ctx)
		if err != nil {
			utils.SendError(resp, err, http.StatusInternalServerError)
			return
		}

		chain2ID := config.GetInt64(config.FlagChain2ChainID)
		if rpcURLChainID.Int64() != chain2ID {
			utils.SendError(resp, fmt.Errorf("URL: %s chainID missmatch - expected: %d but got: %d", rpc, chain2ID, rpcURLChainID), http.StatusBadRequest)
			return
		}
	}
	resp.WriteHeader(http.StatusOK)
}

// AddRoutesForValidator register /validation endpoint
func AddRoutesForValidator(e *gin.Engine) error {
	validatorEndpoints := &validationEndpoints{}
	e.POST("/validation/validate-rpc-chain2-urls", validatorEndpoints.ValidateRPCChain2URLS)
	return nil
}
