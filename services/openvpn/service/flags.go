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

package service

import (
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/urfave/cli"
)

// Options describes options which are required to start Openvpn service
type Options struct {
	OpenvpnProtocol string
	OpenvpnPort     int
}

var (
	protocolFlag = cli.StringFlag{
		Name:  "openvpn.proto",
		Usage: "Openvpn protocol to use. Options: { udp, tcp }",
		Value: "udp",
	}
	portFlag = cli.IntFlag{
		Name:  "openvpn.port",
		Usage: "Openvpn port to use. Default 1194",
		Value: 1194,
	}
)

// RegisterFlags function register Openvpn flags to flag list
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, protocolFlag, portFlag)
}

// ParseFlags function fills in Openvpn options from CLI context
func ParseFlags(ctx *cli.Context) service.Options {
	return Options{
		OpenvpnProtocol: ctx.String(protocolFlag.Name),
		OpenvpnPort:     ctx.Int(portFlag.Name),
	}
}
