//+build !windows

/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

// MaxResources sets the limit to the maximum number of wireguard connections.
const MaxResources = 256

// Allocator is mock wireguard resource handler.
// It will manage lists of network interfaces names, IP addresses and port for endpoints.
type Allocator struct {
	mu          sync.Mutex
	Ifaces      map[int]struct{}
	IPAddresses map[int]struct{}
	Ports       map[int]struct{}

	portMin int
	portMax int
	subnet  net.IPNet
}

// NewAllocator creates new resource pool for wireguard connection.
func NewAllocator(portMin, portMax int, subnet net.IPNet) *Allocator {
	return &Allocator{
		Ifaces:      make(map[int]struct{}),
		IPAddresses: make(map[int]struct{}),
		Ports:       make(map[int]struct{}),

		portMin: portMin,
		portMax: portMax,
		subnet:  subnet,
	}
}

// AbandonedInterfaces returns a list of abandoned interfaces that exist in the system,
// but was not allocated by the Allocator.
func (a *Allocator) AbandonedInterfaces() ([]net.Interface, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	list := make([]net.Interface, 0)
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, interfacePrefix) {
			ifaceID, err := strconv.Atoi(strings.TrimPrefix(iface.Name, interfacePrefix))
			if err == nil {
				if _, ok := a.Ifaces[ifaceID]; !ok {
					list = append(list, iface)
				}
			}
		}
	}

	return list, nil
}

// AllocateInterface provides available name for the wireguard network interface.
func (a *Allocator) AllocateInterface() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for i := 0; i < MaxResources; i++ {
		if _, ok := a.Ifaces[i]; !ok {
			a.Ifaces[i] = struct{}{}
			if interfaceExists(ifaces, fmt.Sprintf("%s%d", interfacePrefix, i)) {
				continue
			}

			return fmt.Sprintf("%s%d", interfacePrefix, i), nil
		}
	}

	return "", errors.New("no more unused interfaces")
}

// AllocateIPNet provides available IP address for the wireguard connection.
func (a *Allocator) AllocateIPNet() (net.IPNet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := 0; i < MaxResources; i++ {
		if _, ok := a.IPAddresses[i]; !ok {
			a.IPAddresses[i] = struct{}{}
			return calcIPNet(a.subnet, i), nil
		}
	}
	return net.IPNet{}, errors.New("no more unused subnets")
}

// AllocatePort provides available UDP port for the wireguard endpoint.
func (a *Allocator) AllocatePort() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := a.portMin; i < a.portMax; i++ {
		if _, ok := a.Ports[i]; !ok {
			a.Ports[i] = struct{}{}
			return i, nil
		}
	}

	return 0, errors.New("no more unused ports")
}

// ReleaseInterface releases name for the wireguard network interface.
func (a *Allocator) ReleaseInterface(iface string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	i, err := strconv.Atoi(strings.TrimPrefix(iface, interfacePrefix))
	if err != nil {
		return err
	}

	if _, ok := a.Ifaces[i]; !ok {
		return errors.New("allocated interface not found")
	}

	delete(a.Ifaces, i)
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

	i := int(ip4[2])
	if _, ok := a.IPAddresses[i]; !ok {
		return errors.New("allocated subnet not found")
	}

	delete(a.IPAddresses, i)
	return nil
}

// ReleasePort releases UDP port.
func (a *Allocator) ReleasePort(port int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.Ports[port]; !ok {
		return errors.New("allocated port not found")
	}

	delete(a.Ports, port)
	return nil
}

func interfaceExists(ifaces []net.Interface, name string) bool {
	for _, iface := range ifaces {
		if iface.Name == name {
			return true
		}
	}
	return false
}

func calcIPNet(ipnet net.IPNet, index int) net.IPNet {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)
	ip = ip.To4()
	ip[2] = byte(index)
	return net.IPNet{IP: ip, Mask: net.IPv4Mask(255, 255, 255, 0)}
}
