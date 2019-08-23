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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
