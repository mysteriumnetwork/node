/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type l2EthClientURLProvider interface {
	CurrentClientOrder() []string
}

type ethInfoEndpoint struct {
	ethL2URLProvider l2EthClientURLProvider
}

func newEthInfoEndpoint(ethClient l2EthClientURLProvider) *ethInfoEndpoint {
	return &ethInfoEndpoint{ethL2URLProvider: ethClient}
}

// EthInfo provides info about active eth client
// swagger:operation GET /eth-info
// ---
// summary: ETH Client Info
// description: Provides active info about eth client
// response:
//	200:
//		schema:
//			"$ref": "#/definitions/EthInfoResponse"
func (e ethInfoEndpoint) EthInfo(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var response contract.EthInfoResponse
	clientURLOrder := e.ethL2URLProvider.CurrentClientOrder()
	if len(clientURLOrder) != 0 {
		// first one is actively in use
		response.EthRPCL2URL = clientURLOrder[0]
	}
	utils.WriteAsJSON(response, resp)
}

// AddRoutesForEthInfo register /eth-info endpoint
func AddRoutesForEthInfo(
	router *httprouter.Router,
	ethClientURLProvider l2EthClientURLProvider,
) {
	ethInfoEndpoints := newEthInfoEndpoint(ethClientURLProvider)
	router.GET("/eth-info", ethInfoEndpoints.EthInfo)
}
