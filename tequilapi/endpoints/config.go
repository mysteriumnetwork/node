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
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type configProvider interface {
	GetUserConfig() map[string]interface{}
}

// swagger:model configResponse
type configResponse struct {
	// example: {"data":{"access-policy":{"list":"mysterium"},"openvpn":{"port":5522}}}
	Data map[string]interface{} `json:"data"`
}

type configAPI struct {
	config configProvider
}

func newConfigAPI(config configProvider) *configAPI {
	return &configAPI{config: config}
}

// GetUserConfig returns current user configuration
// swagger:operation GET /user/config Configuration object
// ---
// summary: Returns current user configuration
// description: Returns current user configuration
// responses:
//   200:
//     description: User configuration
//     schema:
//       "$ref": "#/definitions/configResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) GetUserConfig(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	res := configResponse{Data: api.config.GetUserConfig()}
	utils.WriteAsJSON(res, writer)
}

// AddRoutesForConfig registers /config endpoints in Tequilapi
func AddRoutesForConfig(
	router *httprouter.Router,
) {
	api := newConfigAPI(config.Current)
	router.GET("/config/user", api.GetUserConfig)
}
