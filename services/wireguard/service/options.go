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

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/rs/zerolog/log"
)

// Options describes options which are required to start Wireguard service
type Options struct {
	ConnectDelay int
	Ports        *port.Range
	Subnet       net.IPNet
}

// WireguardConfiguredOptions returns effective Wireguard service options from configuration
func WireguardConfiguredOptions() Options {
	_, ipnet, err := net.ParseCIDR(config.Current.GetString(config.WireguardListenSubnet.Name))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse subnet option, using default value")
		ipnet = &config.WireguardDefaultOptions.Subnet
	}

	portRange, err := port.ParseRange(config.Current.GetString(config.WireguardListenPorts.Name))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse listen port range, using default value")
		portRange = port.UnspecifiedRange()
	}
	if portRange.Capacity() > resources.MaxConnections {
		log.Warn().Msgf("Specified port range exceeds maximum number of connections allowed for the platform (%d), "+
			"using default value", resources.MaxConnections)
		portRange = port.UnspecifiedRange()
	}
	return Options{
		ConnectDelay: config.Current.GetInt(config.WireguardConnectDelayFlag.Name),
		Ports:        portRange,
		Subnet:       *ipnet,
	}
}

// ParseJSONOptions function fills in Wireguard options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	var requestOptions = WireguardConfiguredOptions()
	if request == nil {
		return requestOptions, nil
	}

	opts := config.WireguardDefaultOptions
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
