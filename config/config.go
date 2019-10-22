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

package config

import (
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/utils/jsonutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

// Topic returns event bus topic for the given config key to listen for its updates
func Topic(configKey string) string {
	return "config:" + configKey
}

// Config stores app configuration: default values + user configuration + CLI flags
type Config struct {
	userConfigLocation string
	defaults           map[string]interface{}
	user               map[string]interface{}
	cli                map[string]interface{}
	eventBus           eventbus.EventBus
}

// Current global configuration instance
var Current *Config

func init() {
	Current = NewConfig()
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		userConfigLocation: "",
		defaults:           make(map[string]interface{}),
		user:               make(map[string]interface{}),
		cli:                make(map[string]interface{}),
	}
}

func (cfg *Config) userConfigLoaded() bool {
	return cfg.userConfigLocation != ""
}

// EnableEventPublishing enables config event publishing to the event bus
func (cfg *Config) EnableEventPublishing(eb eventbus.EventBus) {
	cfg.eventBus = eb
}

// LoadUserConfig loads and remembers user config location
func (cfg *Config) LoadUserConfig(location string) error {
	log.Debug().Msg("Loading user configuration: " + location)
	cfg.userConfigLocation = location
	_, err := toml.DecodeFile(cfg.userConfigLocation, &cfg.user)
	if err != nil {
		return errors.Wrap(err, "failed to decode configuration file")
	}
	cfgJson, err := jsonutil.ToJson(cfg.user)
	if err != nil {
		return err
	}
	log.Info().Msg("User configuration loaded: \n" + cfgJson)
	return nil
}

// SaveUserConfig saves user configuration to the file from which it was loaded
func (cfg *Config) SaveUserConfig() error {
	log.Info().Msg("Saving user configuration")
	if !cfg.userConfigLoaded() {
		return errors.New("user configuration cannot be saved, because it must be loaded first")
	}
	var out strings.Builder
	err := toml.NewEncoder(&out).Encode(cfg.user)
	if err != nil {
		return errors.Wrap(err, "failed to write configuration as toml")
	}
	err = ioutil.WriteFile(cfg.userConfigLocation, []byte(out.String()), 0700)
	if err != nil {
		return errors.Wrap(err, "failed to write configuration to file")
	}
	cfgJson, err := jsonutil.ToJson(cfg.user)
	if err != nil {
		return err
	}
	log.Info().Msg("User configuration written: \n" + cfgJson)
	return nil
}

// GetUserConfig returns user configuration
func (cfg *Config) GetUserConfig() map[string]interface{} {
	return cfg.user
}

// SetDefault sets default value for key
func (cfg *Config) SetDefault(key string, value interface{}) {
	cfg.set(&cfg.defaults, key, value)
}

// SetUser sets user configuration value for key
func (cfg *Config) SetUser(key string, value interface{}) {
	if cfg.eventBus != nil {
		cfg.eventBus.Publish(Topic(key), value)
	}
	cfg.set(&cfg.user, key, value)
}

// SetCLI sets value passed via CLI flag for key
func (cfg *Config) SetCLI(key string, value interface{}) {
	cfg.set(&cfg.cli, key, value)
}

// RemoveUser removes user configuration value for key
func (cfg *Config) RemoveUser(key string) {
	cfg.remove(&cfg.user, key)
}

// RemoveCLI removes configured CLI flag value by key
func (cfg *Config) RemoveCLI(key string) {
	cfg.remove(&cfg.cli, key)
}

// set internal method for setting value in a certain configuration value map
func (cfg *Config) set(configMap *map[string]interface{}, key string, value interface{}) {
	key = strings.ToLower(key)
	segments := strings.Split(key, ".")

	lastKey := strings.ToLower(segments[len(segments)-1])
	deepestMap := deepSearch(*configMap, segments[0:len(segments)-1])

	// set innermost value
	deepestMap[lastKey] = value
}

// remove internal method for removing a configured value in a certain configuration map
func (cfg *Config) remove(configMap *map[string]interface{}, key string) {
	key = strings.ToLower(key)
	segments := strings.Split(key, ".")

	lastKey := strings.ToLower(segments[len(segments)-1])
	deepestMap := deepSearch(*configMap, segments[0:len(segments)-1])

	// set innermost value
	delete(deepestMap, lastKey)
}

// Get gets stored config value as-is
func (cfg *Config) Get(key string) interface{} {
	segments := strings.Split(strings.ToLower(key), ".")
	cliValue := cfg.searchMap(cfg.cli, segments)
	if cliValue != nil {
		log.Debug().Msgf("Returning CLI value %v:%v", key, cliValue)
		return cliValue
	}
	userValue := cfg.searchMap(cfg.user, segments)
	if userValue != nil {
		log.Debug().Msgf("Returning user config value %v:%v", key, userValue)
		return userValue
	}
	defaultValue := cfg.searchMap(cfg.defaults, segments)
	log.Debug().Msgf("Returning default value %v:%v", key, defaultValue)
	return defaultValue
}

// GetInt gets config value as int
func (cfg *Config) GetInt(key string) int {
	return cast.ToInt(cfg.Get(key))
}

// GetString gets config value as string
func (cfg *Config) GetString(key string) string {
	return cast.ToString(cfg.Get(key))
}

// GetBool gets config value as bool
func (cfg *Config) GetBool(key string) bool {
	return cast.ToBool(cfg.Get(key))
}
