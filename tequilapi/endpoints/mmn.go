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
	"net/url"
	"strings"

	"github.com/mysteriumnetwork/node/tequilapi/sso"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/mmn"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

const defaultTequilaLogin = "myst"
const defaultTequilaPass = "mystberry"

type mmnProvider interface {
	GetString(key string) string
	SetUser(key string, value interface{})
	RemoveUser(key string)
	SaveUserConfig() error
}

type mmnAPI struct {
	config        mmnProvider
	mmn           *mmn.MMN
	ssoMystnodes  *sso.Mystnodes
	authenticator authenticator
}

func newMMNAPI(config mmnProvider, client *mmn.MMN, ssoMystnodes *sso.Mystnodes, authenticator authenticator) *mmnAPI {
	return &mmnAPI{config: config, mmn: client, ssoMystnodes: ssoMystnodes, authenticator: authenticator}
}

// GetApiKey returns MMN's API key
// swagger:operation GET /mmn/report MMN getApiKey
//
//	---
//	summary: returns MMN's API key
//	description: returns MMN's API key
//	responses:
//	  200:
//	    description: MMN's API key
//	    schema:
//	      "$ref": "#/definitions/MMNApiKeyRequest"
func (api *mmnAPI) GetApiKey(c *gin.Context) {
	res := contract.MMNApiKeyRequest{ApiKey: api.config.GetString(config.FlagMMNAPIKey.Name)}
	utils.WriteAsJSON(res, c.Writer)
}

// SetApiKey sets MMN's API key
// swagger:operation POST /mmn/api-key MMN setApiKey
//
//	---
//	summary: sets MMN's API key
//	description: sets MMN's API key
//	parameters:
//	  - in: body
//	    name: body
//	    description: api_key field
//	    schema:
//	      $ref: "#/definitions/MMNApiKeyRequest"
//	responses:
//	  200:
//	    description: API key has been set
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  422:
//	    description: Unable to process the request at this point
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *mmnAPI) SetApiKey(c *gin.Context) {
	var req contract.MMNApiKeyRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	api.config.SetUser(config.FlagMMNAPIKey.Name, req.ApiKey)
	if err = api.config.SaveUserConfig(); err != nil {
		c.Error(apierror.Internal("Failed to save API key", contract.ErrCodeConfigSave))
		return
	}

	err = api.mmn.ClaimNode()
	if err != nil {
		log.Error().Msgf("MMN registration error: %s", err.Error())

		switch {
		case strings.Contains(err.Error(), "authentication needed: password or unlock"):
			c.Error(apierror.Unprocessable("Identity is locked", contract.ErrCodeIDLocked))
		case strings.Contains(err.Error(), "already owned"):
			msg := fmt.Sprintf("This node has already been claimed. Please visit %s and unclaim it first.", api.config.GetString(config.FlagMMNAddress.Name))
			c.Error(apierror.Unprocessable(msg, contract.ErrCodeMMNNodeAlreadyClaimed))
		case strings.Contains(err.Error(), "invalid api key"):
			c.Error(apierror.Unprocessable("Invalid API key", contract.ErrCodeMMNAPIKey))
		default:
			c.Error(apierror.Internal("Failed to register to MMN", contract.ErrCodeMMNRegistration))
		}
		return
	}
}

// ClearApiKey clears MMN's API key from config
// swagger:operation DELETE /mmn/api-key MMN clearApiKey
//
//	---
//	summary: Clears MMN's API key from config
//	description: Clears MMN's API key from config
//	responses:
//	  200:
//	    description: MMN API key removed
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *mmnAPI) ClearApiKey(c *gin.Context) {
	api.config.RemoveUser(config.FlagMMNAPIKey.Name)
	if err := api.config.SaveUserConfig(); err != nil {
		c.Error(apierror.Internal("Failed to clear API key", contract.ErrCodeConfigSave))
		return
	}
}

// GetClaimLink generates claim link for mystnodes.com
// swagger:operation GET /mmn/claim-link MMN getClaimLink
//
//	---
//
//	summary: Generate claim link
//	description: Generates claim link to claim Node on mystnodes.com with a click
//	responses:
//	  200:
//	    description: Link response
//	    schema:
//	      "$ref": "#/definitions/MMNLinkRedirectResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *mmnAPI) GetClaimLink(c *gin.Context) {
	redirectQuery := c.Query("redirect_uri")

	if redirectQuery == "" {
		c.Error(apierror.BadRequest("redirect_uri missing", contract.ErrCodeMMNClaimRedirectURLMissing))
		return
	}

	redirectURL, err := url.Parse(redirectQuery)
	if err != nil {
		c.Error(apierror.BadRequest("redirect_uri malformed", contract.ErrCodeMMNClaimRedirectURLMissing))
		return
	}

	link, err := api.mmn.ClaimLink(redirectURL)
	if err != nil {
		c.Error(apierror.Internal("Failed to generate claim link", contract.ErrCodeMMNClaimLink))
		return
	}

	c.JSON(http.StatusOK, &contract.MMNLinkRedirectResponse{Link: link.String()})
}

func (api *mmnAPI) isDefaultCredentials() bool {
	err := api.authenticator.CheckCredentials(defaultTequilaLogin, defaultTequilaPass)
	return err == nil
}

func (api *mmnAPI) isApiKeySet() bool {
	apiKey := api.config.GetString(api.config.GetString(config.FlagMMNAPIKey.Name))
	return apiKey != ""
}

// GetOnboardingLink generates claim link for mystnodes.com
// swagger:operation GET /mmn/onboarding MMN GetOnboardingLink
//
//	---
//
//	summary: Generate onboarding link
//	description: Generates onboarding link for one click onboarding
//	responses:
//	  200:
//	    description: Link response
//	    schema:
//	      "$ref": "#/definitions/MMNLinkRedirectResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *mmnAPI) GetOnboardingLink(c *gin.Context) {
	if !api.isDefaultCredentials() || api.isApiKeySet() {
		c.Error(apierror.Unauthorized())
		return
	}

	redirectQuery := c.Query("redirect_uri")

	if redirectQuery == "" {
		c.Error(apierror.BadRequest("redirect_uri missing", contract.ErrCodeMMNClaimRedirectURLMissing))
		return
	}

	redirectURL, err := url.Parse(redirectQuery)
	if err != nil {
		c.Error(apierror.BadRequest("redirect_uri malformed", contract.ErrCodeMMNClaimRedirectURLMissing))
		return
	}

	link, err := api.ssoMystnodes.SSOLink(redirectURL)
	link.Path = "/clickboarding"
	if err != nil {
		c.Error(apierror.Internal("Failed to generate claim link", contract.ErrCodeMMNClaimLink))
		return
	}

	c.JSON(http.StatusOK, &contract.MMNLinkRedirectResponse{Link: link.String()})
}

// VerifyGrant verify grant
// swagger:operation POST /mmn/onboarding MMN VerifyGrant
//
//	---
//
//	summary: verify grant for onboarding
//	description: verify grant for onboarding
//	responses:
//	  200:
//	    description: Link response
//	    schema:
//	      "$ref": "#/definitions/MMNGrantVerificationResponse"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (api *mmnAPI) VerifyGrant(c *gin.Context) {
	if !api.isDefaultCredentials() || api.isApiKeySet() {
		c.Error(apierror.Unauthorized())
		return
	}

	var request contract.MystnodesSSOGrantLoginRequest
	err := json.NewDecoder(c.Request.Body).Decode(&request)
	if err != nil {
		log.Err(err).Msg("failed decoding request")
		c.Error(apierror.Unauthorized())
		return
	}

	vi, err := api.ssoMystnodes.VerifyAuthorizationGrant(request.AuthorizationGrant, sso.VerificationOptions{SkipNodeClaimCheck: true})
	if err != nil {
		log.Err(err).Msg("failed to verify mystnodes Authorization Grant")
		c.Error(apierror.Unauthorized())
		return
	}

	r := contract.MMNGrantVerificationResponse{
		ApiKey:                        vi.APIkey,
		WalletAddress:                 vi.WalletAddress,
		IsEligibleForFreeRegistration: vi.IsEligibleForFreeRegistration,
	}

	c.JSON(http.StatusOK, r)
}

// AddRoutesForMMN registers /mmn endpoints in Tequilapi
func AddRoutesForMMN(
	mmn *mmn.MMN,
	ssoMystnodes *sso.Mystnodes,
	authenticator authenticator,
) func(*gin.Engine) error {
	api := newMMNAPI(config.Current, mmn, ssoMystnodes, authenticator)
	return func(e *gin.Engine) error {
		g := e.Group("/mmn")
		{
			g.GET("/api-key", api.GetApiKey)
			g.POST("/api-key", api.SetApiKey)
			g.DELETE("/api-key", api.ClearApiKey)

			g.GET("/claim-link", api.GetClaimLink)

			g.GET("/onboarding", api.GetOnboardingLink)
			g.POST("/onboarding/verify-grant", api.VerifyGrant)
		}
		return nil
	}
}
