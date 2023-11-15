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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const configFile = "myst_supervisor.conf"

// Config for supervisor, created during -install.
type Config struct {
	Uid string
}

func (c Config) valid() bool {
	return c.Uid != ""
}

// Write config file.
func (c Config) Write() error {
	if !c.valid() {
		return errors.New("configuration is not valid")
	}
	confPath, err := configPath()
	if err != nil {
		return err
	}

	var out strings.Builder
	err = toml.NewEncoder(&out).Encode(c)
	if err != nil {
		return fmt.Errorf("could not encode configuration: %w", err)
	}
	if err := os.WriteFile(confPath, []byte(out.String()), 0700); err != nil {
		return fmt.Errorf("could not write %q: %w", confPath, err)
	}
	return nil
}

// Read config file.
func Read() (*Config, error) {
	confPath, err := configPath()
	if err != nil {
		return nil, err
	}

	c := Config{}
	_, err = toml.DecodeFile(confPath, &c)
	if err != nil {
		return nil, fmt.Errorf("could not read %q: %w", confPath, err)
	}
	if !c.valid() {
		return nil, fmt.Errorf("invalid configuration file %q, please re-install the supervisor (-install)", confPath)
	}
	return &c, nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config dir: %w", err)
	}
	return filepath.Join(dir, configFile), nil
}
