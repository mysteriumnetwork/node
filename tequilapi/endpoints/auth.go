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
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"

	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type authenticationAPI struct {
	jwtAuthenticator jwtAuthenticator
	authenticator    authenticator
}

type jwtAuthenticator interface {
	CreateToken(username string) (auth.JWT, error)
}

type authenticator interface {
	CheckCredentials(username, password string) error
	ChangePassword(username, oldPassword, newPassword string) error
}

// swagger:operation POST /auth/authenticate Authentication Authenticate
// ---
// summary: Authenticate
// description: Authenticates user and issues auth token
// parameters:
//   - in: body
//     name: body
//     schema:
//       $ref: "#/definitions/AuthRequest"
// responses:
//   200:
//     description: Authentication succeeded
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   401:
//     description: Authentication failed
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *authenticationAPI) Authenticate(httpRes http.ResponseWriter, httpReq *http.Request, _ httprouter.Params) {
	req, err := toAuthRequest(httpReq)
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
		utils.SendError(httpRes, err, http.StatusInternalServerError)
		return
	}

	response := contract.NewAuthResponse(jwtToken)
	utils.WriteAsJSON(response, httpRes)
}

// swagger:operation POST /auth/login Authentication Login
// ---
// summary: Login
// description: Authenticates user and sets cookie with issued auth token
// parameters:
//   - in: body
//     name: body
//     schema:
//       $ref: "#/definitions/AuthRequest"
// responses:
//   200:
//     description: Authentication succeeded
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   401:
//     description: Authentication failed
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *authenticationAPI) Login(httpRes http.ResponseWriter, httpReq *http.Request, _ httprouter.Params) {
	req, err := toAuthRequest(httpReq)
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
		utils.SendError(httpRes, err, http.StatusInternalServerError)
		return
	}

	response := contract.NewAuthResponse(jwtToken)

	http.SetCookie(httpRes, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    jwtToken.Token,
		Expires:  jwtToken.ExpirationTime,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
	})
	utils.WriteAsJSON(response, httpRes)
}

// swagger:operation DELETE /auth/logout Authentication Logout
// ---
// summary: Logout
// description: Clears authentication cookie
// responses:
//   200:
//     description: Logged out successfully
func (api *authenticationAPI) Logout(httpRes http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	http.SetCookie(httpRes, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   0,
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
	var req *contract.ChangePasswordRequest
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

func toAuthRequest(req *http.Request) (contract.AuthRequest, error) {
	var request contract.AuthRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	return request, err
}

func toChangePasswordRequest(req *http.Request) (*contract.ChangePasswordRequest, error) {
	var cpReq = contract.ChangePasswordRequest{}
	if err := json.NewDecoder(req.Body).Decode(&cpReq); err != nil {
		return nil, err
	}
	return &cpReq, nil
}

// TequilapiAuthenticateEndpointPath used by UIServer to know which endpoint doesn't need auth
const TequilapiAuthenticateEndpointPath = "/auth/authenticate"

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
	router.POST(TequilapiAuthenticateEndpointPath, api.Authenticate)
	router.POST(TequilapiLoginEndpointPath, api.Login)
	router.DELETE("/auth/logout", api.Logout)
}
