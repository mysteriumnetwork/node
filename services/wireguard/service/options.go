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
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/urfave/cli"
)

// Options describes options which are required to start Wireguard service
type Options struct {
	ConnectDelay   int       `json:"connectDelay"`
	PortMin        int       `json:"portMin"`
	MaxConnections int       `json:"maxConnections"`
	Subnet         net.IPNet `json:"subnet"`
}

var (
	delayFlag = cli.IntFlag{
		Name:  "wireguard.connect.delay",
		Usage: "Consumer is delayed by specified time if provider is behind NAT",
		Value: DefaultOptions.ConnectDelay,
	}
	portMin = cli.IntFlag{
		Name:  "wireguard.listen.port.min",
		Usage: "Min value of the allowed range of listen ports",
		Value: DefaultOptions.PortMin,
	}
	maxConnections = cli.IntFlag{
		Name:  "wireguard.listen.max-connections",
		Usage: "Maximum number of connections",
		Value: DefaultOptions.MaxConnections,
	}
	subnet = cli.StringFlag{
		Name:  "wireguard.allowed.subnet",
		Usage: "Subnet allowed for using by the wireguard services",
		Value: DefaultOptions.Subnet.String(),
	}
)

// RegisterFlags function register Wireguard flags to flag list
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, delayFlag, portMin, maxConnections, subnet)
}

// ParseFlags function fills in Wireguard options from CLI context
func ParseFlags(ctx *cli.Context) service.Options {
	_, ipnet, err := net.ParseCIDR(ctx.String(subnet.Name))
	if err != nil {
		log.Warn(logPrefix, "Failed to parse subnet option, using default value. ", err)
		ipnet = &DefaultOptions.Subnet
	}

	return Options{
		ConnectDelay:   ctx.Int(delayFlag.Name),
		PortMin:        ctx.Int(portMin.Name),
		MaxConnections: ctx.Int(maxConnections.Name),
		Subnet:         *ipnet,
	}
}

// ParseJSONOptions function fills in Openvpn options from JSON request
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
		ConnectDelay   int    `json:"connectDelay"`
		PortMin        int    `json:"portMin"`
		MaxConnections int    `json:"maxConnections"`
		Subnet         string `json:"subnet"`
	}{
		ConnectDelay:   o.ConnectDelay,
		PortMin:        o.PortMin,
		MaxConnections: o.MaxConnections,
		Subnet:         o.Subnet.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (o *Options) UnmarshalJSON(data []byte) error {
	var options struct {
		ConnectDelay   int    `json:"connectDelay"`
		PortMin        int    `json:"portMin"`
		MaxConnections int    `json:"maxConnections"`
		Subnet         string `json:"subnet"`
	}

	if err := json.Unmarshal(data, &options); err != nil {
		return err
	}

	if options.ConnectDelay != 0 {
		o.ConnectDelay = options.ConnectDelay
	}
	if options.PortMin != 0 {
		o.PortMin = options.PortMin
	}
	if options.MaxConnections != 0 {
		o.MaxConnections = options.MaxConnections
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
