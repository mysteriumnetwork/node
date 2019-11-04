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
	"net"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/services/wireguard/service"
	"gopkg.in/urfave/cli.v1"
)

var (
	WireguardConnectDelayFlag = cli.IntFlag{
		Name:  "wireguard.connect.delay",
		Usage: "Consumer is delayed by specified time if provider is behind NAT",
		Value: WireguardDefaultOptions.ConnectDelay,
	}
	WireguardListenPorts = cli.StringFlag{
		Name:  "wireguard.listen.ports",
		Usage: "Range of listen ports (e.g. 52820:53075)",
	}
	WireguardListenSubnet = cli.StringFlag{
		Name:  "wireguard.allowed.subnet",
		Usage: "Subnet allowed for using by the wireguard services",
		Value: WireguardDefaultOptions.Subnet.String(),
	}
	// WireguardDefaultOptions is a wireguard service configuration that will be used if no options provided.
	WireguardDefaultOptions = service.Options{
		ConnectDelay: 2000,
		Ports:        port.UnspecifiedRange(),
		Subnet: net.IPNet{
			IP:   net.ParseIP("10.182.0.0"),
			Mask: net.IPv4Mask(255, 255, 0, 0),
		},
	}
)

// RegisterFlagsServiceWireguard function register Wireguard flags to flag list
func RegisterFlagsServiceWireguard(flags *[]cli.Flag) {
	*flags = append(*flags, WireguardConnectDelayFlag, WireguardListenPorts, WireguardListenSubnet)
}

// ParseFlagsServiceWireguard parses CLI flags and registers value to configuration
func ParseFlagsServiceWireguard(ctx *cli.Context) {
	Current.SetDefault(WireguardConnectDelayFlag.Name, WireguardDefaultOptions.ConnectDelay)
	Current.SetDefault(WireguardListenPorts.Name, WireguardDefaultOptions.Ports)
	Current.SetDefault(WireguardListenSubnet.Name, WireguardDefaultOptions.Subnet)
	SetIntFlag(Current, WireguardConnectDelayFlag.Name, ctx)
	SetStringFlag(Current, WireguardListenPorts.Name, ctx)
	SetStringFlag(Current, WireguardListenSubnet.Name, ctx)
}
