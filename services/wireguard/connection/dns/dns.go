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

package dns

import (
	"errors"
	"sync"
)

// Manager is connection DNS configuration manager.
type Manager interface {
	// Set applies DNS configuration.
	Set(cfg Config) error
	// Clean removes DNS configuration.
	Clean() error
}

// Config represent dns manager config.
type Config struct {
	// ScriptDir used only for unix.
	ScriptDir string
	IfaceName string
	DNS       []string
}

// NewManager returns new DNS manager instance.
func NewManager() Manager {
	return &dnsManager{}
}

type dnsManager struct {
	mu  sync.Mutex
	cfg *Config
}

func (dm *dnsManager) Set(cfg Config) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.cfg = &cfg
	// Empty DNS means that system DNS will be used, no need to set anything.
	if len(cfg.DNS) == 0 {
		return nil
	}

	return setDNS(cfg)
}

func (dm *dnsManager) Clean() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.cfg == nil {
		return errors.New("DNS is already cleaned")
	}

	if len(dm.cfg.DNS) == 0 {
		dm.cfg = nil
		return nil
	}

	err := cleanDNS(*dm.cfg)
	dm.cfg = nil
	return err
}
