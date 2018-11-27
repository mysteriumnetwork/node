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

import "net"

// Handler is mock wireguard resource handler.
// It will manage lists of network interfaces names, IP addresses and port for endpoints.
type Handler struct{}

// AllocateInterface provides available name for the wireguard network interface.
func (h *Handler) AllocateInterface() string { return "myst0" }

// AllocateIP provides available IP address for the wireguard connection.
func (h *Handler) AllocateIP() net.IPNet {
	ip, ipnet, _ := net.ParseCIDR("10.182.47.1/24")
	return net.IPNet{IP: ip, Mask: ipnet.Mask}
}

// AllocatePort provides available UDP port for the wireguard endpoint.
func (h *Handler) AllocatePort() int { return 52820 }

// ReleaseInterface releases name for the wireguard network interface.
func (h *Handler) ReleaseInterface(string) error { return nil }

// ReleaseIP releases IP address.
func (h *Handler) ReleaseIP(ip string) error { return nil }

// ReleasePort releases UDP port.
func (h *Handler) ReleasePort(port int) error { return nil }
