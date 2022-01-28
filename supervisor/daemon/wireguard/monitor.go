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

package wireguard

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/supervisor/daemon/wireguard/wginterface"
)

// Monitor creates/deletes the WireGuard interfaces and keeps track of them.
type Monitor struct {
	interfaces map[string]*wginterface.WgInterface
	mu         sync.Mutex
}

// NewMonitor creates new WireGuard monitor instance.
func NewMonitor() *Monitor {
	return &Monitor{
		interfaces: make(map[string]*wginterface.WgInterface),
	}
}

// Up requests interface creation.
func (m *Monitor) Up(cfg wgcfg.DeviceConfig, uid string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if iface, exists := m.interfaces[cfg.IfaceName]; exists {
		if err := iface.Reconfigure(cfg); err != nil {
			return "", fmt.Errorf("failed to reconfigure interface %s: %w", cfg.IfaceName, err)
		}
		return iface.Name, nil
	}

	iface, err := wginterface.New(cfg, uid)
	if err != nil {
		return "", err
	}

	m.interfaces[iface.Name] = iface
	return iface.Name, err
}

// Down requests interface deletion.
func (m *Monitor) Down(interfaceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	iface, ok := m.interfaces[interfaceName]
	if !ok {
		return fmt.Errorf("interface %s not found", interfaceName)
	}

	iface.Down()
	delete(m.interfaces, interfaceName)
	return nil
}

// Stats requests interface statistics.
func (m *Monitor) Stats(interfaceName string) (wgcfg.Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	iface, ok := m.interfaces[interfaceName]
	if !ok {
		return wgcfg.Stats{}, fmt.Errorf("interface %s not found", interfaceName)
	}

	deviceState, err := userspace.ParseUserspaceDevice(iface.Device.IpcGetOperation)
	if err != nil {
		return wgcfg.Stats{}, fmt.Errorf("could not parse device state: %w", err)
	}

	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(time.Second) {
		stats, statErr := userspace.ParseDevicePeerStats(deviceState)
		if err != nil {
			err = statErr
			log.Warn().Err(err).Msg("Failed to parse device stats, will try again")
		} else {
			return stats, nil
		}
	}

	return wgcfg.Stats{}, err
}
