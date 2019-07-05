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

	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type authenticationAPI struct {
	jwtAuthenticator jwtAuthenticator
	authenticator    authenticator
}

type jwtAuthenticator interface {
	CreateToken(username string) (auth.JWTToken, error)
}

type authenticator interface {
	CheckCredentials(username, password string) error
	ChangePassword(username, oldPassword, newPassword string) error
}

// swagger:model LoginRequest
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// swagger:model ChangePasswordRequest
type changePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// swagger:operation POST /auth/login Authentication Login
// ---
// summary: Login
// description: Checks user credentials and sets JWT session cookie
// parameters:
//   - in: body
//     name: body
//     schema:
//       $ref: "#/definitions/LoginRequest"
// responses:
//   200:
//     description: Logged in successfully
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   401:
//     description: Unauthorized
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *authenticationAPI) Login(httpRes http.ResponseWriter, httpReq *http.Request, _ httprouter.Params) {
	var req *loginRequest
	var err error

	req, err = toLoginRequest(httpReq)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusBadRequest)
		return
	}
	err = api.authenticator.CheckCredentials(req.Username, req.Password)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusUnauthorized)
		return
	}

	jwtToken, err := api.jwtAuthenticator.CreateToken(req.Username)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusBadRequest)
		return
	}

	http.SetCookie(httpRes, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    jwtToken.Token,
		Expires:  jwtToken.ExpirationTime,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
	})
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
	err = api.authenticator.ChangePassword(req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		utils.SendError(httpRes, err, http.StatusUnauthorized)
		return
	}
}

func toLoginRequest(req *http.Request) (*loginRequest, error) {
	var loginReq = loginRequest{}
	if err := json.NewDecoder(req.Body).Decode(&loginReq); err != nil {
		return nil, err
	}
	return &loginReq, nil
}

func toChangePasswordRequest(req *http.Request) (*changePasswordRequest, error) {
	var cpReq = changePasswordRequest{}
	if err := json.NewDecoder(req.Body).Decode(&cpReq); err != nil {
		return nil, err
	}
	return &cpReq, nil
}

// TequilapiLoginEndpointPath used by UIServer to know which endpoint doesn't need auth
const TequilapiLoginEndpointPath = "/auth/login"

// AddRoutesForAuthentication registers /auth endpoints in Tequilapi
func AddRoutesForAuthentication(
	router *httprouter.Router,
	auth authenticator,
	jwtAuth jwtAuthenticator,
) {
	api := &authenticationAPI{
		authenticator:    auth,
		jwtAuthenticator: jwtAuth,
	}
	router.PUT("/auth/password", api.ChangePassword)
	router.POST(TequilapiLoginEndpointPath, api.Login)
}
