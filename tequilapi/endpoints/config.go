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

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/rs/zerolog/log"
)

type configProvider interface {
	GetConfig() map[string]interface{}
	GetDefaultConfig() map[string]interface{}
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

// GetConfig returns current configuration
// swagger:operation GET /config Configuration getConfig
// ---
// summary: Returns current configuration values
// description: Returns default configuration
// responses:
//   200:
//     description: Currently active configuration
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) GetConfig(c *gin.Context) {
	writer := c.Writer
	res := configPayload{Data: api.config.GetConfig()}
	utils.WriteAsJSON(res, writer)
}

// GetDefaultConfig returns default configuration
// swagger:operation GET /config/default Configuration getDefaultConfig
// ---
// summary: Returns default configuration
// description: Returns default configuration
// responses:
//   200:
//     description: Default configuration values
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) GetDefaultConfig(c *gin.Context) {
	writer := c.Writer
	res := configPayload{Data: api.config.GetDefaultConfig()}
	utils.WriteAsJSON(res, writer)
}

// GetUserConfig returns current user configuration
// swagger:operation GET /config/user Configuration getUserConfig
// ---
// summary: Returns current user configuration
// description: Returns current user configuration
// responses:
//   200:
//     description: User set configuration values
//     schema:
//       "$ref": "#/definitions/configPayload"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (api *configAPI) GetUserConfig(c *gin.Context) {
	writer := c.Writer
	res := configPayload{Data: api.config.GetUserConfig()}
	utils.WriteAsJSON(res, writer)
}

// SetUserConfig sets and returns current configuration
// swagger:operation POST /config/user Configuration serUserConfig
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
func (api *configAPI) SetUserConfig(c *gin.Context) {
	httpReq := c.Request
	writer := c.Writer

	var req configPayload
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}
	for k, v := range req.Data {
		if isNil(v) {
			log.Debug().Msgf("Clearing user config value: %q", v)
			api.config.RemoveUser(k)
		} else {
			log.Debug().Msgf("Setting user config value: %q = %q", k, v)
			api.config.SetUser(k, v)
		}
	}
	err = api.config.SaveUserConfig()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}
	api.GetUserConfig(c)
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
	e *gin.Engine,
) error {
	api := newConfigAPI(config.Current)
	g := e.Group("/config")
	{
		g.GET("", api.GetConfig)
		g.GET("/default", api.GetDefaultConfig)
		g.GET("/user", api.GetUserConfig)
		g.POST("/user", api.SetUserConfig)
	}
	return nil
}
