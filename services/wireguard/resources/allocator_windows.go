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

package resources

import (
	"errors"
	"net"
	"sync"
)

// MaxResources sets the limit to the maximum number of wireguard connections.
const MaxResources = 255

// Allocator is mock wireguard resource handler.
// It will manage lists of network interfaces names, IP addresses and port for endpoints.
type Allocator struct {
	IPAddresses map[int]struct{}
	mu          sync.Mutex

	listenPort int
	subnet     net.IPNet
}

// NewAllocator creates new resource pool for wireguard connection.
func NewAllocator(listenPort int, subnet net.IPNet) *Allocator {
	return &Allocator{
		IPAddresses: make(map[int]struct{}),

		listenPort: listenPort,
		subnet:     subnet,
	}
}

// AbandonedInterfaces is not required for Windows implementation and left here just to satisfy the interface.
func (a *Allocator) AbandonedInterfaces() ([]net.Interface, error) {
	return nil, nil
}

// AllocateInterface provides available name for the wireguard network interface.
func (a *Allocator) AllocateInterface() (string, error) {
	return interfacePrefix, nil
}

// AllocateIPNet provides available IP address for the wireguard connection.
func (a *Allocator) AllocateIPNet() (net.IPNet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := 1; i < MaxResources; i++ {
		if _, ok := a.IPAddresses[i]; !ok {
			a.IPAddresses[i] = struct{}{}
			return calcIPNet(a.subnet, i), nil
		}
	}
	return net.IPNet{}, errors.New("no more unused subnets")
}

// AllocatePort provides available UDP port for the wireguard endpoint.
func (a *Allocator) AllocatePort() (int, error) {
	return a.listenPort, nil
}

// ReleaseInterface is not required for Windows implementation and left here just to satisfy the interface.
func (a *Allocator) ReleaseInterface(iface string) error {
	return nil
}

// ReleaseIPNet releases IP address.
func (a *Allocator) ReleaseIPNet(ipnet net.IPNet) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return errors.New("allocated subnet not found")
	}

	i := int(ip4[3])
	if _, ok := a.IPAddresses[i]; !ok {
		return errors.New("allocated subnet not found")
	}

	delete(a.IPAddresses, i)
	return nil
}

// ReleasePort is not required for Windows implementation and left here just to satisfy the interface.
func (a *Allocator) ReleasePort(port int) error {
	return nil
}

func calcIPNet(ipnet net.IPNet, index int) net.IPNet {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)
	ip = ip.To4()
	ip[3] = byte(index)
	return net.IPNet{IP: ip, Mask: net.IPv4Mask(255, 255, 255, 0)}
}
