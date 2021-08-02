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

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
)

// Options describes options which are required to start Wireguard service.
type Options struct {
	Subnet net.IPNet
}

// DefaultOptions is a wireguard service configuration that will be used if no options provided.
var DefaultOptions = Options{
	Subnet: net.IPNet{
		IP:   net.ParseIP("10.182.0.0").To4(),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	},
}

// GetOptions returns effective Wireguard service options from application configuration.
func GetOptions() Options {
	_, ipnet, err := net.ParseCIDR(config.GetString(config.FlagWireguardListenSubnet))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse subnet option, using default value")
		ipnet = &DefaultOptions.Subnet
	}

	return Options{
		Subnet: *ipnet,
	}
}

// ParseJSONOptions function fills in Wireguard options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	requestOptions := GetOptions()
	if request == nil {
		return requestOptions, nil
	}

	opts := DefaultOptions
	err := json.Unmarshal(*request, &opts)
	return opts, err
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (o Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Subnet string `json:"subnet"`
	}{
		Subnet: o.Subnet.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (o *Options) UnmarshalJSON(data []byte) error {
	var options struct {
		Subnet string `json:"subnet"`
	}

	if err := json.Unmarshal(data, &options); err != nil {
		return err
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
