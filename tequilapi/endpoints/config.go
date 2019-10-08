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
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type configProvider interface {
	GetUserConfig() map[string]interface{}
	SetUser(key string, value interface{})
	RemoveUser(key string)
	SaveUserConfig() error
}

// swagger:model configPayload
type configPayload struct {
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
// swagger:operation GET /user/config Configuration getUserConfig
// ---
// summary: Returns current user configuration
// description: Returns current user configuration
// responses:
//   200:
//     description: User configuration
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) GetUserConfig(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	res := configPayload{Data: api.config.GetUserConfig()}
	utils.WriteAsJSON(res, writer)
}

// SetUserConfig sets and returns current configuration
// swagger:operation POST /user/config Configuration serUserConfig
// ---
// summary: Sets and returns user configuration
// description: For keys present in the payload, it will set or remove the user config values (if the key is null). Changes are persisted to the config file.
// parameters:
//   - in: body
//     name: body
//     description: configuration keys/values
//     schema:
//       $ref: "#/definitions/configPayload"
// responses:
//   200:
//     description: User configuration
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) SetUserConfig(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	var req configPayload
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}
	for k, v := range req.Data {
		if isNil(v) {
			log.Debugf("clearing user config value: %q")
			api.config.RemoveUser(k)
		} else {
			log.Debugf("setting user config value: %q = %q", k, v)
			api.config.SetUser(k, v)
		}
	}
	err = api.config.SaveUserConfig()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
	api.GetUserConfig(writer, nil, nil)
}

func isNil(val interface{}) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return true
	}
	return false
}

// AddRoutesForConfig registers /config endpoints in Tequilapi
func AddRoutesForConfig(
	router *httprouter.Router,
) {
	api := newConfigAPI(config.Current)
	router.GET("/config/user", api.GetUserConfig)
	router.POST("/config/user", api.SetUserConfig)
}
