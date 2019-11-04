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
	"github.com/mysteriumnetwork/node/services/openvpn/service"
	"gopkg.in/urfave/cli.v1"
)

var OpenvpnProtocolFlag = cli.StringFlag{
	Name:  "openvpn.proto",
	Usage: "OpenVPN protocol to use. Options: { udp, tcp }",
	//TODO remove value
	Value: "udp",
}

var OpenvpnPortFlag = cli.IntFlag{
	Name:  "openvpn.port",
	Usage: "OpenVPN port to use. If not specified, random port will be used",
	//TODO remove value
	Value: 0,
}

var OpenvpnSubnetFlag = cli.StringFlag{
	Name:  "openvpn.subnet",
	Usage: "OpenVPN subnet that will be used to connecting VPN clients",
}

var OpenvpnNetmaskFlag = cli.StringFlag{
	Name:  "openvpn.netmask",
	Usage: "OpenVPN subnet netmask ",
}

// RegisterFlagsServiceOpenvpn registers OpenVPN CLI flags for parsing them later
func RegisterFlagsServiceOpenvpn(flags *[]cli.Flag) {
	*flags = append(*flags, OpenvpnProtocolFlag, OpenvpnPortFlag, OpenvpnSubnetFlag, OpenvpnNetmaskFlag)
}

// ParseFlagsServiceOpenvpn parses CLI flags and registers value to configuration
func ParseFlagsServiceOpenvpn(ctx *cli.Context) {
	Current.SetDefault(OpenvpnProtocolFlag.Name, DefaultOptionsOpenvpn.Protocol)
	Current.SetDefault(OpenvpnPortFlag.Name, DefaultOptionsOpenvpn.Port)
	Current.SetDefault(OpenvpnSubnetFlag.Name, DefaultOptionsOpenvpn.Subnet)
	Current.SetDefault(OpenvpnNetmaskFlag.Name, DefaultOptionsOpenvpn.Netmask)
	SetStringFlag(Current, OpenvpnProtocolFlag.Name, ctx)
	SetIntFlag(Current, OpenvpnPortFlag.Name, ctx)
	SetStringFlag(Current, OpenvpnSubnetFlag.Name, ctx)
	SetStringFlag(Current, OpenvpnNetmaskFlag.Name, ctx)
}

var (
	DefaultOptionsOpenvpn = service.Options{
		Protocol: "udp",
		Port:     0,
		Subnet:   "10.8.0.0",
		Netmask:  "255.255.255.0",
	}
)
