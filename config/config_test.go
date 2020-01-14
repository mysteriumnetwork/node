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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// This only tests user configuration, not the merging between multiple option sources
func TestUserConfig_Load(t *testing.T) {
	// given
	configFileName := NewTempFileName(t)
	defer func() {
		_ = os.Remove(configFileName)
	}()
	toml := `
		[openvpn]
		port = 31338
	`
	err := ioutil.WriteFile(configFileName, []byte(toml), 0700)
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
	defer func() {
		_ = os.Remove(configFileName)
	}()
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
	tomlContent, err := ioutil.ReadFile(configFileName)
	assert.NoError(t, err)
	assert.Contains(t, string(tomlContent), "port = 22822")
	assert.NotContains(t, string(tomlContent), `proto = "tcp"`)
}

func NewTempFileName(t *testing.T) string {
	file, err := ioutil.TempFile("", "*")
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
				Usage: `Proposal discovery adapter(s) separated by comma Options: { "api", "broker", "api,broker" }`,
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

func must(t *testing.T, err error) {
	assert.NoError(t, err)
}
