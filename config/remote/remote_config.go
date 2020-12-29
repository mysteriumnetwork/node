/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package remoteconfig

import (
	"math/big"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/spf13/cast"

	"github.com/mysteriumnetwork/node/tequilapi/client"

	"github.com/mysteriumnetwork/node/config"
	"github.com/rs/zerolog/log"
)

// Config - to be used in cli, account, connections or any similar action that requires a backing node
var Config = newRemoteConfig()

type remoteConfig struct {
	config map[string]interface{}
}

func newRemoteConfig() *remoteConfig {
	return &remoteConfig{}
}

// RefreshRemoteConfig - will fetch latest config with provided client.Client
func RefreshRemoteConfig(client *client.Client) error {
	config, err := client.FetchConfig()
	if err != nil {
		return err
	}
	Config.config = config
	return nil
}

// Get returns stored remote config value as-is.
func (rc *remoteConfig) Get(key string) interface{} {
	segments := strings.Split(strings.ToLower(key), ".")
	value := config.SearchMap(rc.config, segments)
	log.Debug().Msgf("Returning remote config value %v:%v", key, value)
	return value
}

// GetStringByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *remoteConfig) GetStringByFlag(flag cli.StringFlag) string {
	return rc.GetString(flag.Name)
}

// GetString returns config value as string.
func (rc *remoteConfig) GetString(key string) string {
	return cast.ToString(rc.Get(key))
}

// GetBoolByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *remoteConfig) GetBoolByFlag(flag cli.BoolFlag) bool {
	return rc.GetBool(flag.Name)
}

// GetBool returns config value as bool.
func (rc *remoteConfig) GetBool(key string) bool {
	return cast.ToBool(rc.Get(key))
}

// GetBigIntByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a big.Int.
func (rc *remoteConfig) GetBigIntByFlag(flag cli.StringFlag) *big.Int {
	return rc.GetBigInt(flag.Name)
}

// GetBigInt returns config value as big.Int.
func (rc *remoteConfig) GetBigInt(key string) *big.Int {
	b, _ := new(big.Int).SetString(rc.GetString(key), 10)
	return b
}

// GetStringSliceByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a []string.
func (rc *remoteConfig) GetStringSliceByFlag(flag cli.StringSliceFlag) []string {
	return rc.GetStringSlice(flag.Name)
}

// GetStringSlice returns config value as []string.
func (rc *remoteConfig) GetStringSlice(key string) []string {
	return cast.ToStringSlice(rc.Get(key))
}

// GetInt64ByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a int64.
func (rc *remoteConfig) GetInt64ByFlag(flag cli.Int64Flag) int64 {
	return rc.GetInt64(flag.Name)
}

// GetInt64 returns config value as int64.
func (rc *remoteConfig) GetInt64(key string) int64 {
	return cast.ToInt64(rc.Get(key))
}

// TequilAPIAddress - wil resolve default tequilapi address or from flag if one is provided
func TequilAPIAddress(ctx *cli.Context) string {
	flag := config.FlagTequilapiAddress

	if ctx.IsSet(flag.Name) {
		return ctx.String(flag.Name)
	}

	return flag.Value
}

// TequilAPIPort - wil resolve default tequilapi port or from flag if one is provided
func TequilAPIPort(ctx *cli.Context) int {
	flag := config.FlagTequilapiPort

	if ctx.IsSet(flag.Name) {
		return ctx.Int(flag.Name)
	}

	return flag.Value
}
