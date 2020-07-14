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

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type mmnProvider interface {
	GetString(key string) string
	SetUser(key string, value interface{})
	RemoveUser(key string)
	SaveUserConfig() error
}

// swagger:model configPayload
type mmnPayload struct {
	ApiKey string `json:"api-key"`
}

type mmnAPI struct {
	config mmnProvider
}

func newMMNAPI(config mmnProvider) *mmnAPI {
	return &mmnAPI{config: config}
}

// GetDefaultConfig returns default configuration
// swagger:operation GET /user/default Configuration getDefaultConfig
// ---
// summary: Returns default configuration
// description: Returns default configuration
// responses:
//   200:
//     description: Default configuration
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *mmnAPI) GetMMNReport(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	res := mmnPayload{ApiKey: api.config.GetString("mmn.api-key")}
	utils.WriteAsJSON(res, writer)
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
func (api *mmnAPI) GetMMNConfig(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	res := mmnPayload{ApiKey: api.config.GetString("mmn.api-key")}
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
func (api *mmnAPI) SetMMNConfig(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	var req mmnPayload
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}

	if len(req.ApiKey) == 0 {
		log.Debug().Msgf("clearing MMN config")
		api.config.RemoveUser("mmn")
	} else {
		log.Debug().Msgf("setting MMN config")
		api.config.SetUser("mmn", req)
	}

	err = api.config.SaveUserConfig()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
}

// AddRoutesForConfig registers /config endpoints in Tequilapi
func AddRoutesForMMN(
	router *httprouter.Router,
) {
	api := newMMNAPI(config.Current)
	router.GET("/mmn/report", api.GetMMNReport)
	router.GET("/mmn/config", api.GetMMNConfig)
	router.POST("/mmn/config", api.SetMMNConfig)
}
