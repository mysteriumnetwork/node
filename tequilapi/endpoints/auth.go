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
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/auth"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/sso"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type authenticationAPI struct {
	jwtAuthenticator jwtAuthenticator
	authenticator    authenticator
	ssoMystnodes     *sso.Mystnodes
}

type jwtAuthenticator interface {
	CreateToken(username string) (auth.JWT, error)
}

type authenticator interface {
	CheckCredentials(username, password string) error
	ChangePassword(username, oldPassword, newPassword string) error
}

// swagger:operation POST /auth/authenticate Authentication Authenticate
//
//	---
//	summary: Authenticate
//	description: Authenticates user and issues auth token
//	parameters:
//	  - in: body
//	    name: body
//	    schema:
//	      $ref: "#/definitions/AuthRequest"
//	responses:
//	  200:
//	    description: Authentication succeeded
//	    schema:
//	      "$ref": "#/definitions/AuthResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  401:
//	    description: Authentication failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *authenticationAPI) Authenticate(c *gin.Context) {
	req, err := toAuthRequest(c.Request)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}
	err = api.authenticator.CheckCredentials(req.Username, req.Password)
	if err != nil {
		c.Error(apierror.Unauthorized())
		return
	}

	jwtToken, err := api.jwtAuthenticator.CreateToken(req.Username)
	if err != nil {
		c.Error(apierror.Unauthorized())
		return
	}

	response := contract.NewAuthResponse(jwtToken)
	utils.WriteAsJSON(response, c.Writer)
}

// swagger:operation POST /auth/login Authentication Login
//
//	---
//	summary: Login
//	description: Authenticates user and sets cookie with issued auth token
//	parameters:
//	  - in: body
//	    name: body
//	    schema:
//	      $ref: "#/definitions/AuthRequest"
//	responses:
//	  200:
//	    description: Authentication succeeded
//	    schema:
//	      "$ref": "#/definitions/AuthResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  401:
//	    description: Authentication failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *authenticationAPI) Login(c *gin.Context) {
	req, err := toAuthRequest(c.Request)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}
	err = api.authenticator.CheckCredentials(req.Username, req.Password)
	if err != nil {
		c.Error(apierror.Unauthorized())
		return
	}

	jwtToken, err := api.jwtAuthenticator.CreateToken(req.Username)
	if err != nil {
		c.Error(apierror.Unauthorized())
		return
	}

	response := contract.NewAuthResponse(jwtToken)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    jwtToken.Token,
		Expires:  jwtToken.ExpirationTime,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
	})
	utils.WriteAsJSON(response, c.Writer)
}

// swagger:operation GET /auth/login-mystnodes SSO LoginMystnodesInit
//
//	---
//	summary: LoginMystnodesInit
//	description: SSO init endpoint to auth via mystnodes
//	parameters:
//	  - in: query
//	    name: redirect_url
//	    description: a redirect to send authorization grant to
//	    type: string
//
//	responses:
//	  200:
//	    description: link response
//	    schema:
//	      "$ref": "#/definitions/MystnodesSSOLinkResponse"
func (api *authenticationAPI) LoginMystnodesInit(c *gin.Context) {
	redirectURL, err := url.Parse(c.Query("redirect_uri"))
	if err != nil {
		log.Err(err).Msg("failed to generate mystnodes SSO link")
		c.Error(apierror.Unauthorized())
		return
	}

	link, err := api.ssoMystnodes.SSOLink(redirectURL)
	if err != nil {
		log.Err(err).Msg("failed to generate mystnodes SSO link")
		c.Error(apierror.Unauthorized())
		return
	}
	c.JSON(http.StatusOK, contract.MystnodesSSOLinkResponse{Link: link.String()})
}

// swagger:operation POST /auth/login-mystnodes SSO LoginMystnodesWithGrant
//
//	---
//	summary: LoginMystnodesWithGrant
//	description: SSO login using grant provided by mystnodes.com
//
//	responses:
//	  200:
//	    description: grant was verified against mystnodes using PKCE workflow. This will set access token cookie.
//	  401:
//	    description: grant failed to be verified
func (api *authenticationAPI) LoginMystnodesWithGrant(c *gin.Context) {
	var request contract.MystnodesSSOGrantLoginRequest
	err := json.NewDecoder(c.Request.Body).Decode(&request)
	if err != nil {
		log.Err(err).Msg("failed decoding request")
		c.Error(apierror.Unauthorized())
		return
	}

	_, err = api.ssoMystnodes.VerifyAuthorizationGrant(request.AuthorizationGrant, sso.DefaultVerificationOptions)
	if err != nil {
		log.Err(err).Msg("failed to verify mystnodes Authorization Grant")
		c.Error(apierror.Unauthorized())
		return
	}

	jwtToken, err := api.jwtAuthenticator.CreateToken("myst")
	if err != nil {
		c.Error(apierror.Unauthorized())
		return
	}

	response := contract.NewAuthResponse(jwtToken)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    jwtToken.Token,
		Expires:  jwtToken.ExpirationTime,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
	})
	utils.WriteAsJSON(response, c.Writer)
}

// swagger:operation DELETE /auth/logout Authentication Logout
//
//	---
//	summary: Logout
//	description: Clears authentication cookie
//	responses:
//	  200:
//	    description: Logged out successfully
func (api *authenticationAPI) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
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
//
//	---
//	summary: Change password
//	description: Changes user password
//	parameters:
//	  - in: body
//	    name: body
//	    schema:
//	      $ref: "#/definitions/ChangePasswordRequest"
//	responses:
//	  200:
//	    description: Password changed successfully
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  401:
//	    description: Unauthorized
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *authenticationAPI) ChangePassword(c *gin.Context) {
	var req *contract.ChangePasswordRequest
	var err error
	req, err = toChangePasswordRequest(c.Request)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}
	err = api.authenticator.ChangePassword(req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		c.Error(apierror.Unauthorized())
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

// AddRoutesForAuthentication registers /auth endpoints in Tequilapi
func AddRoutesForAuthentication(auth authenticator, jwtAuth jwtAuthenticator, ssoMystnodes *sso.Mystnodes) func(*gin.Engine) error {
	api := &authenticationAPI{
		authenticator:    auth,
		jwtAuthenticator: jwtAuth,
		ssoMystnodes:     ssoMystnodes,
	}
	return func(e *gin.Engine) error {
		g := e.Group("/auth")
		{
			g.PUT("/password", api.ChangePassword)
			g.POST("/authenticate", api.Authenticate)
			g.POST("/login", api.Login)
			g.GET("/login-mystnodes", api.LoginMystnodesInit)
			g.POST("/login-mystnodes", api.LoginMystnodesWithGrant)
			g.DELETE("/logout", api.Logout)
		}
		return nil
	}
}
