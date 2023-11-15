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
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/scraping"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// This only tests user configuration, not the merging between multiple option sources
func TestUserConfig_Load(t *testing.T) {
	// given
	configFileName := NewTempFileName(t)
	defer os.Remove(configFileName)

	toml := `
		[openvpn]
		port = 31338
	`
	err := os.WriteFile(configFileName, []byte(toml), 0700)
	assert.NoError(t, err)

	// when
	cfg := NewConfig()
	// then
	assert.Nil(t, cfg.Get("openvpn.port"))

	// when
	err = cfg.LoadUserConfig(configFileName)
	// then
	assert.NoError(t, err)
	assert.Equal(t, 31338, cfg.GetInt("openvpn.port"))
}

func TestUserConfig_Save(t *testing.T) {
	// given
	configFileName := NewTempFileName(t)
	defer os.Remove(configFileName)

	cfg := NewConfig()
	err := cfg.LoadUserConfig(configFileName)
	assert.NoError(t, err)

	// when: app is configured with defaults + user + CLI values
	cfg.SetDefault("openvpn.proto", "tcp")
	cfg.SetDefault("openvpn.port", 55)
	cfg.SetUser("openvpn.port", 22822)
	cfg.SetCLI("openvpn.port", 40000)
	// then: CLI values are prioritized over user over defaults
	assert.Equal(t, "tcp", cfg.GetString("openvpn.proto"))
	assert.Equal(t, 40000, cfg.GetInt("openvpn.port"))

	// when: user configuration is saved
	err = cfg.SaveUserConfig()
	// then: only user configuration values are stored
	assert.NoError(t, err)
	tomlContent, err := os.ReadFile(configFileName)
	assert.NoError(t, err)
	assert.Contains(t, string(tomlContent), "port = 22822")
	assert.NotContains(t, string(tomlContent), `proto = "tcp"`)
}

func NewTempFileName(t *testing.T) string {
	file, err := os.CreateTemp("", "*")
	assert.NoError(t, err)
	return file.Name()
}

func TestConfig_ParseStringSliceFlag(t *testing.T) {
	var tests = []struct {
		name     string
		args     string
		defaults *cli.StringSlice
		values   []string
	}{
		{
			name:     "parse single value",
			defaults: cli.NewStringSlice("api", "broker"),
			args:     "--discovery.type broker",
			values:   []string{"broker"},
		},
		{
			name:     "parse multiple values",
			defaults: cli.NewStringSlice("api", "broker"),
			args:     "--discovery.type api --discovery.type broker",
			values:   []string{"api", "broker"},
		},
		{
			name:     "empty args use defaults",
			defaults: cli.NewStringSlice("api", "broker"),
			args:     "",
			values:   []string{"api", "broker"},
		},
		{
			name:   "nil default returns empty slice",
			args:   "",
			values: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			sliceFlag := cli.StringSliceFlag{
				Name:  "discovery.type",
				Usage: `Proposal discovery adapter(s) separated by comma. Options: { "api", "broker", "api,broker" }`,
				Value: tc.defaults,
			}
			cfg := NewConfig()
			flagSet := flag.NewFlagSet("", flag.ContinueOnError)
			must(t, sliceFlag.Apply(flagSet))
			ctx := cli.NewContext(nil, flagSet, nil)
			must(t, flagSet.Parse(strings.Split(tc.args, " ")))

			// when
			cfg.ParseStringSliceFlag(ctx, sliceFlag)
			value := cfg.GetStringSlice(sliceFlag.Name)

			// then
			assert.Equal(t, tc.values, value)
		})
	}
}

// this can happen when updating config via tequilapi - json unmarshal
// translates json number to float64 by default if target type is interface{}
func TestSimilarTypeMerge(t *testing.T) {
	// given
	cfg := NewConfig()
	cfg.SetDefault("openvpn.port", 1001)

	// when
	cfg.SetUser("openvpn.port", 55.00)

	// then
	assert.Equal(t, 55, cfg.GetInt("openvpn.port"))

	actual, ok := cfg.GetConfig()["openvpn"]
	assert.True(t, ok)

	actual, ok = actual.(map[string]interface{})["port"]
	assert.True(t, ok)

	assert.Equal(t, 55.0, actual)
}

func TestUserConfig_Get(t *testing.T) {
	cfg := NewConfig()

	// when
	cfg.SetDefault("openvpn.port", 1001)
	// then
	assert.Equal(t, 1001, cfg.Get("openvpn.port"))

	// when
	cfg.SetUser("openvpn.port", 1002)
	// then
	assert.Equal(t, 1002, cfg.Get("openvpn.port"))

	// when
	cfg.SetCLI("openvpn.port", 1003)
	// then
	assert.Equal(t, 1003, cfg.Get("openvpn.port"))
}

func TestUserConfig_GetConfig(t *testing.T) {
	cfg := NewConfig()

	// when
	cfg.SetDefault("enabled", false)
	cfg.SetDefault("openvpn.port", 1001)
	// then
	assert.Equal(
		t,
		map[string]interface{}{
			"enabled": false,
			"openvpn": map[string]interface{}{
				"port": 1001,
			},
		},
		cfg.GetConfig(),
	)

	// when
	cfg.SetUser("openvpn.port", 1002)
	// then
	assert.Equal(
		t,
		map[string]interface{}{
			"enabled": false,
			"openvpn": map[string]interface{}{
				"port": 1002,
			},
		},
		cfg.GetConfig(),
	)

	// when
	cfg.SetCLI("enabled", true)
	cfg.SetCLI("openvpn.port", 1003)
	// then
	assert.Equal(
		t,
		map[string]interface{}{
			"enabled": true,
			"openvpn": map[string]interface{}{
				"port": 1003,
			},
		},
		cfg.GetConfig(),
	)
}

func TestHardcodedServicesNameFlagValues(t *testing.T) {
	// importing these constants into config package create cyclic dependency
	assert.Equal(t, strings.Join([]string{scraping.ServiceType, datatransfer.ServiceType, dvpn.ServiceType}, ","), FlagActiveServices.Value)
}

func must(t *testing.T, err error) {
	assert.NoError(t, err)
}
