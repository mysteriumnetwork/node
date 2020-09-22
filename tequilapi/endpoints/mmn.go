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
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

type mmnProvider interface {
	GetString(key string) string
	SetUser(key string, value interface{})
	RemoveUser(key string)
	SaveUserConfig() error
}

type mmnAPI struct {
	config mmnProvider
	mmn    *mmn.MMN
}

func newMMNAPI(config mmnProvider, client *mmn.MMN) *mmnAPI {
	return &mmnAPI{config: config, mmn: client}
}

// GetNodeReport returns node report from MMN
// swagger:operation GET /mmn/node-report MMN getNodeReport
// ---
// summary: Returns node report from MMN
// description: Returns node report from MMN
// responses:
//   200:
//     description: Node report from MMN
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *mmnAPI) GetNodeReport(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	report, err := api.mmn.GetReport()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-type", "application/json; charset=utf-8")
	fmt.Fprint(writer, report)
}

// GetApiKey returns MMN's API key
// swagger:operation GET /mmn/report MMN getApiKey
// ---
// summary: returns MMN's API key
// description: returns MMN's API key
// responses:
//   200:
//     description: MMN's API key
//     schema:
//       "$ref": "#/definitions/MMNApiKeyRequest"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *mmnAPI) GetApiKey(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	res := contract.MMNApiKeyRequest{ApiKey: api.config.GetString(config.FlagMMNKey.Name)}
	utils.WriteAsJSON(res, writer)
}

// SetApiKey sets MMN's API key
// swagger:operation POST /mmn/api-key MMN setApiKey
// ---
// summary: sets MMN's API key
// description: sets MMN's API key
// parameters:
//   - in: body
//     name: body
//     description: api_key field
//     schema:
//       $ref: "#/definitions/MMNApiKeyRequest"
// responses:
//   200:
//     description: Success
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"

func (api *mmnAPI) SetApiKey(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	var req contract.MMNApiKeyRequest
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}

	if errors := req.Validate(); errors.HasErrors() {
		utils.SendValidationErrorMessage(writer, errors)
		return
	}

	api.config.SetUser(config.FlagMMNKey.Name, req.ApiKey)
	if err = api.config.SaveUserConfig(); err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	err = api.mmn.Register()
	if err != nil {
		log.Error().Msgf("MMN registration error: %s", err.Error())

		switch {
		case strings.Contains(err.Error(), "authentication needed: password or unlock"):
			errors := validation.NewErrorMap()
			errors.ForField("api_key").AddError("identity", "Identity is locked")
			utils.SendValidationErrorMessage(writer, errors)
		case strings.Contains(err.Error(), "already owned"):
			errors := validation.NewErrorMap()
			errors.ForField("api_key").AddError(
				"already_owned",
				"This node has already been claimed. Please visit https://my.mysterium.network/ and unclaim it first.",
			)
			utils.SendValidationErrorMessage(writer, errors)
		case strings.Contains(err.Error(), "invalid api key"):
			errors := validation.NewErrorMap()
			errors.ForField("api_key").AddError("not_found", "Invalid API key")
			utils.SendValidationErrorMessage(writer, errors)
		default:
			utils.SendError(writer, err, http.StatusInternalServerError)
		}
		return
	}
}

// ClearApiKey clears MMN's API key from config
// swagger:operation DELETE /mmn/api-key MMN clearApiKey
// ---
// summary: Clears MMN's API key from config
// description: Clears MMN's API key from config
// responses:
//   200:
//     description: MMN's API key removed
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *mmnAPI) ClearApiKey(writer http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	api.config.RemoveUser(config.FlagMMNKey.Name)
	if err := api.config.SaveUserConfig(); err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
}

// AddRoutesForMMN registers /mmn endpoints in Tequilapi
func AddRoutesForMMN(
	router *httprouter.Router,
	mmn *mmn.MMN,
) {
	api := newMMNAPI(config.Current, mmn)
	router.GET("/mmn/report", api.GetNodeReport)
	router.GET("/mmn/api-key", api.GetApiKey)
	router.POST("/mmn/api-key", api.SetApiKey)
	router.DELETE("/mmn/api-key", api.ClearApiKey)
}
