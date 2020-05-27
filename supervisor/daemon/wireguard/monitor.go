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
func (m *Monitor) Up(requestedInterfaceName string, uid string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.interfaces[requestedInterfaceName]; exists {
		return "", fmt.Errorf("interface %s already exists", requestedInterfaceName)
	}
	iface, err := wginterface.New(requestedInterfaceName, uid)
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
