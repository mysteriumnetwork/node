/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/tequilapi/utils"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/rs/zerolog/log"
)

type termsAPI struct {
	config configProvider
}

func newTermsAPI(config configProvider) *termsAPI {
	return &termsAPI{config: config}
}

// GetTerms returns current terms config
//
// swagger:operation GET /terms Terms getTerms
// ---
// summary: Get terms
// description: Return an object with the current terms config
// responses:
//   200:
//     description: Terms object
//     schema:
//       "$ref": "#/definitions/TermsResponse"
func (api *termsAPI) GetTerms(c *gin.Context) {
	c.JSON(http.StatusOK, contract.NewTermsResp())
}

// UpdateTerms accepts new terms and updates user config
//
// swagger:operation POST /terms Terms updateTerms
// ---
// summary: Update terms agreement
// description: Takes the given data and tries to update terms agreement config.
// parameters:
// - in: body
//   name: body
//   description: Required data to update terms
//   schema:
//     $ref: "#/definitions/TermsRequest"
// responses:
//   200:
//     description: OK
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *termsAPI) UpdateTerms(c *gin.Context) {
	r := c.Request
	w := c.Writer

	var req contract.TermsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.SendError(w, err, http.StatusBadRequest)
		return
	}

	for k, v := range req.ToMap() {
		log.Debug().Msgf("Setting user config value: %q = %q", k, v)
		api.config.SetUser(k, v)
	}

	err = api.config.SaveUserConfig()
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// AddRoutesForTerms registers /terms endpoints in Tequilapi
func AddRoutesForTerms(e *gin.Engine) error {
	api := newTermsAPI(config.Current)

	g := e.Group("/terms")
	g.GET("", api.GetTerms)
	g.POST("", api.UpdateTerms)
	return nil
}
