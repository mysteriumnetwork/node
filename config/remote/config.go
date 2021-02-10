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

package remote

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/mysteriumnetwork/node/config"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"
)

// Config - remote config struct
type Config struct {
	client Fetcher
	config map[string]interface{}
}

// Fetcher interface represents anything that
// is able to fetch a config.
type Fetcher interface {
	FetchConfig() (map[string]interface{}, error)
}

// NewConfig - new remote config instance
func NewConfig(client Fetcher) (*Config, error) {
	cfg := &Config{
		client: client,
	}
	return cfg, cfg.RefreshRemoteConfig()
}

// RefreshRemoteConfig - will fetch latest config
func (rc *Config) RefreshRemoteConfig() error {
	config, err := rc.client.FetchConfig()
	if err != nil {
		return err
	}
	rc.config = config
	return nil
}

// Get returns stored remote config value as-is.
func (rc *Config) Get(key string) interface{} {
	segments := strings.Split(strings.ToLower(key), ".")
	value := config.SearchMap(rc.config, segments)
	log.Debug().Msgf("Returning remote config value %v:%v", key, value)
	return value
}

// GetStringByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *Config) GetStringByFlag(flag cli.StringFlag) string {
	return rc.GetString(flag.Name)
}

// GetString returns config value as string.
func (rc *Config) GetString(key string) string {
	return cast.ToString(rc.Get(key))
}

// GetBoolByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *Config) GetBoolByFlag(flag cli.BoolFlag) bool {
	return rc.GetBool(flag.Name)
}

// GetBool returns config value as bool.
func (rc *Config) GetBool(key string) bool {
	return cast.ToBool(rc.Get(key))
}

// GetBigIntByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a big.Int.
func (rc *Config) GetBigIntByFlag(flag cli.StringFlag) *big.Int {
	return rc.GetBigInt(flag.Name)
}

// GetBigInt returns config value as big.Int.
func (rc *Config) GetBigInt(key string) *big.Int {
	b, _ := new(big.Int).SetString(rc.GetString(key), 10)
	return b
}

// GetStringSliceByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a []string.
func (rc *Config) GetStringSliceByFlag(flag cli.StringSliceFlag) []string {
	return rc.GetStringSlice(flag.Name)
}

// GetStringSlice returns config value as []string.
func (rc *Config) GetStringSlice(key string) []string {
	return cast.ToStringSlice(rc.Get(key))
}

// GetInt64ByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a int64.
func (rc *Config) GetInt64ByFlag(flag cli.Int64Flag) int64 {
	return rc.GetInt64(flag.Name)
}

// GetInt64 returns config value as int64.
func (rc *Config) GetInt64(key string) int64 {
	return cast.ToInt64(rc.Get(key))
}

// GetHermesID returns the current hermes id.
func (rc *Config) GetHermesID() (string, error) {
	chid := rc.GetInt64ByFlag(config.FlagChainID)
	if chid == rc.GetInt64ByFlag(config.FlagChain1ChainID) {
		return rc.GetStringByFlag(config.FlagChain1HermesAddress), nil
	}

	if chid == rc.GetInt64ByFlag(config.FlagChain2ChainID) {
		return rc.GetStringByFlag(config.FlagChain2HermesAddress), nil
	}

	return "", fmt.Errorf("no hermes specified for chain %v", chid)
}
