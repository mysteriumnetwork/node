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
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/utils/jsonutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"
)

// Config stores application configuration in 3 separate maps (listed from the lowest priority to the highest):
//
// • Default values
//
// • User configuration (config.toml)
//
// • CLI flags
type Config struct {
	userConfigLocation string
	defaults           map[string]interface{}
	user               map[string]interface{}
	cli                map[string]interface{}
	eventBus           eventbus.EventBus
}

// Current global configuration instance.
var Current *Config

func init() {
	Current = NewConfig()
}

// NewConfig creates a new configuration instance.
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

// EnableEventPublishing enables config event publishing to the event bus.
func (cfg *Config) EnableEventPublishing(eb eventbus.EventBus) {
	cfg.eventBus = eb
}

// LoadUserConfig loads and remembers user config location.
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

// SaveUserConfig saves user configuration to the file from which it was loaded.
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

// GetUserConfig returns user configuration.
func (cfg *Config) GetUserConfig() map[string]interface{} {
	return cfg.user
}

// SetDefault sets default value for key.
func (cfg *Config) SetDefault(key string, value interface{}) {
	cfg.set(&cfg.defaults, key, value)
}

// SetUser sets user configuration value for key.
func (cfg *Config) SetUser(key string, value interface{}) {
	if cfg.eventBus != nil {
		cfg.eventBus.Publish(AppTopicConfig(key), value)
	}
	cfg.set(&cfg.user, key, value)
}

// SetCLI sets value passed via CLI flag for key.
func (cfg *Config) SetCLI(key string, value interface{}) {
	cfg.set(&cfg.cli, key, value)
}

// RemoveUser removes user configuration value for key.
func (cfg *Config) RemoveUser(key string) {
	cfg.remove(&cfg.user, key)
}

// RemoveCLI removes configured CLI flag value by key.
func (cfg *Config) RemoveCLI(key string) {
	cfg.remove(&cfg.cli, key)
}

// set sets value to a particular configuration value map.
func (cfg *Config) set(configMap *map[string]interface{}, key string, value interface{}) {
	key = strings.ToLower(key)
	segments := strings.Split(key, ".")

	lastKey := strings.ToLower(segments[len(segments)-1])
	deepestMap := deepSearch(*configMap, segments[0:len(segments)-1])

	// set innermost value
	deepestMap[lastKey] = value
}

// remove removes a configured value from a particular configuration map.
func (cfg *Config) remove(configMap *map[string]interface{}, key string) {
	key = strings.ToLower(key)
	segments := strings.Split(key, ".")

	lastKey := strings.ToLower(segments[len(segments)-1])
	deepestMap := deepSearch(*configMap, segments[0:len(segments)-1])

	// set innermost value
	delete(deepestMap, lastKey)
}

// Get returns stored config value as-is.
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

// GetBool returns config value as bool.
func (cfg *Config) GetBool(key string) bool {
	return cast.ToBool(cfg.Get(key))
}

// GetInt returns config value as int.
func (cfg *Config) GetInt(key string) int {
	return cast.ToInt(cfg.Get(key))
}

// GetUInt64 returns config value as uint64.
func (cfg *Config) GetUInt64(key string) uint64 {
	return cast.ToUint64(cfg.Get(key))
}

// GetFloat64 returns config value as float64.
func (cfg *Config) GetFloat64(key string) float64 {
	return cast.ToFloat64(cfg.Get(key))
}

// GetDuration returns config value as duration.
func (cfg *Config) GetDuration(key string) time.Duration {
	return cast.ToDuration(cfg.Get(key))
}

// GetString returns config value as string.
func (cfg *Config) GetString(key string) string {
	return cast.ToString(cfg.Get(key))
}

// GetStringSlice returns config value as []string.
func (cfg *Config) GetStringSlice(key string) []string {
	value := cfg.Get(key).([]string)
	return cast.ToStringSlice(value)
}

// ParseBoolFlag parses a cli.BoolFlag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseBoolFlag(ctx *cli.Context, flag cli.BoolFlag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.Bool(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseIntFlag parses a cli.IntFlag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseIntFlag(ctx *cli.Context, flag cli.IntFlag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.Int(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseUInt64Flag parses a cli.Uint64Flag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseUInt64Flag(ctx *cli.Context, flag cli.Uint64Flag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.Uint64(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseFloat64Flag parses a cli.Float64Flag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseFloat64Flag(ctx *cli.Context, flag cli.Float64Flag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.Float64(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseDurationFlag parses a cli.DurationFlag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseDurationFlag(ctx *cli.Context, flag cli.DurationFlag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.Duration(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseStringFlag parses a cli.StringFlag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseStringFlag(ctx *cli.Context, flag cli.StringFlag) {
	cfg.SetDefault(flag.Name, flag.Value)
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.String(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// ParseStringSliceFlag parses a cli.StringSliceFlag from command's context and
// sets default and CLI values to the application configuration.
func (cfg *Config) ParseStringSliceFlag(ctx *cli.Context, flag cli.StringSliceFlag) {
	cfg.SetDefault(flag.Name, flag.Value.Value())
	if ctx.IsSet(flag.Name) {
		cfg.SetCLI(flag.Name, ctx.StringSlice(flag.Name))
	} else {
		cfg.RemoveCLI(flag.Name)
	}
}

// GetBool shorthand for getting current configuration value for cli.BoolFlag.
func GetBool(flag cli.BoolFlag) bool {
	return Current.GetBool(flag.Name)
}

// GetInt shorthand for getting current configuration value for cli.IntFlag.
func GetInt(flag cli.IntFlag) int {
	return Current.GetInt(flag.Name)
}

// GetString shorthand for getting current configuration value for cli.StringFlag.
func GetString(flag cli.StringFlag) string {
	return Current.GetString(flag.Name)
}

// GetStringSlice shorthand for getting current configuration value for cli.StringSliceFlag.
func GetStringSlice(flag cli.StringSliceFlag) []string {
	return Current.GetStringSlice(flag.Name)
}

// GetDuration shorthand for getting current configuration value for cli.DurationFlag.
func GetDuration(flag cli.DurationFlag) time.Duration {
	return Current.GetDuration(flag.Name)
}

// GetUInt64 shorthand for getting current configuration value for cli.Uint64Flag.
func GetUInt64(flag cli.Uint64Flag) uint64 {
	return Current.GetUInt64(flag.Name)
}

// GetFloat64 shorthand for getting current configuration value for cli.Uint64Flag.
func GetFloat64(flag cli.Float64Flag) float64 {
	return Current.GetFloat64(flag.Name)
}

// AppTopicConfig returns event bus topic for the given config key to listen for its updates.
func AppTopicConfig(configKey string) string {
	return "config:" + configKey
}
