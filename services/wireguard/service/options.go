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

package service

import (
	"encoding/json"
	"net"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/urfave/cli.v1/altsrc"
)

// Options describes options which are required to start Wireguard service
type Options struct {
	ConnectDelay int
	Ports        *port.Range
	Subnet       net.IPNet
}

var (
	delayFlag = altsrc.NewIntFlag(cli.IntFlag{
		Name:  "wireguard.connect.delay",
		Usage: "Consumer is delayed by specified time if provider is behind NAT",
		Value: DefaultOptions.ConnectDelay,
	})
	ports = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "wireguard.listen.ports",
		Usage: "Range of listen ports (e.g. 52820:53075)",
	})
	subnet = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "wireguard.allowed.subnet",
		Usage: "Subnet allowed for using by the wireguard services",
		Value: DefaultOptions.Subnet.String(),
	})
)

// DefaultOptions is a wireguard service configuration that will be used if no options provided.
var DefaultOptions = Options{
	ConnectDelay: 2000,
	Ports:        port.UnspecifiedRange(),
	Subnet: net.IPNet{
		IP:   net.ParseIP("10.182.0.0"),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}}

// RegisterFlags function register Wireguard flags to flag list
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, delayFlag, ports, subnet)
}

// ParseFlags function fills in Wireguard options from CLI context
func ParseFlags(ctx *cli.Context) service.Options {
	_, ipnet, err := net.ParseCIDR(ctx.String(subnet.Name))
	if err != nil {
		log.Warn(logPrefix, "Failed to parse subnet option, using default value. ", err)
		ipnet = &DefaultOptions.Subnet
	}

	portRange, err := port.ParseRange(ctx.String(ports.Name))
	if err != nil {
		log.Warn(logPrefix, "Failed to parse listen port range, using default value. ", err)
		portRange = port.UnspecifiedRange()
	}
	if portRange.Capacity() > resources.MaxConnections {
		log.Warnf(logPrefix, "Specified port range exceeds maximum number of connections allowed for the platform (%d), "+
			"using default value", resources.MaxConnections)
		portRange = port.UnspecifiedRange()
	}

	return Options{
		ConnectDelay: ctx.Int(delayFlag.Name),
		Ports:        portRange,
		Subnet:       *ipnet,
	}
}

// ParseJSONOptions function fills in Wireguard options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	if request == nil {
		return DefaultOptions, nil
	}

	opts := DefaultOptions
	err := json.Unmarshal(*request, &opts)
	return opts, err
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (o Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ConnectDelay int    `json:"connectDelay"`
		Ports        string `json:"ports"`
		Subnet       string `json:"subnet"`
	}{
		ConnectDelay: o.ConnectDelay,
		Ports:        o.Ports.String(),
		Subnet:       o.Subnet.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (o *Options) UnmarshalJSON(data []byte) error {
	var options struct {
		ConnectDelay int    `json:"connectDelay"`
		Ports        string `json:"ports"`
		Subnet       string `json:"subnet"`
	}

	if err := json.Unmarshal(data, &options); err != nil {
		return err
	}

	if options.ConnectDelay != 0 {
		o.ConnectDelay = options.ConnectDelay
	}
	if options.Ports != "" {
		p, err := port.ParseRange(options.Ports)
		if err != nil {
			return err
		}
		o.Ports = p
	}
	if len(options.Subnet) > 0 {
		_, ipnet, err := net.ParseCIDR(options.Subnet)
		if err != nil {
			return err
		}
		o.Subnet = *ipnet
	}

	return nil
}
