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
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type authenticationAPI struct {
	passwordChanger passwordChanger
}

type passwordChanger interface {
	ChangePassword(username, oldPassword, newPassword string) error
}

// swagger:model ChangePasswordRequest
type changePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// swagger:operation PUT /auth/password Authentication changePassword
// ---
// summary: Change password
// description: Changes user password
// parameters:
//   - in: body
//     name: body
//     schema:
//       $ref: "#/definitions/ChangePasswordRequest"
// responses:
//   200:
//     description: Password changed successfully
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   401:
//     description: Unauthorized
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *authenticationAPI) ChangePassword(httpRes http.ResponseWriter, httpReq *http.Request, _ httprouter.Params) {
	var req *changePasswordRequest
	var err error
	req, err = toChangePasswordRequest(httpReq)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusBadRequest)
		return
	}
	err = api.passwordChanger.ChangePassword(req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusUnauthorized)
		return
	}
}

func toChangePasswordRequest(req *http.Request) (*changePasswordRequest, error) {
	var cpReq = changePasswordRequest{}
	if err := json.NewDecoder(req.Body).Decode(&cpReq); err != nil {
		return nil, err
	}
	return &cpReq, nil
}

// AddRoutesForAuthentication registers /auth endpoints in Tequilapi
func AddRoutesForAuthentication(
	router *httprouter.Router,
	passwordChanger passwordChanger,
) {
	api := &authenticationAPI{
		passwordChanger: passwordChanger,
	}
	router.PUT("/auth/password", api.ChangePassword)
}
