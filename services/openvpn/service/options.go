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
	"encoding/json"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/rs/zerolog/log"
)

// Options describes options which are required to start Openvpn service
type Options struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Subnet   string `json:"subnet"`
	Netmask  string `json:"netmask"`
}

// GetOptions returns effective OpenVPN service options from application configuration.
func GetOptions() Options {
	return Options{
		Protocol: config.GetString(config.FlagOpenvpnProtocol),
		Port:     config.GetInt(config.FlagOpenvpnPort),
		Subnet:   config.GetString(config.FlagOpenvpnSubnet),
		Netmask:  config.GetString(config.FlagOpenvpnNetmask),
	}
}

// ParseJSONOptions function fills in OpenVPN options from JSON request, falling back to configured options for
// missing values
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	var requestOptions = GetOptions()
	if request == nil {
		return requestOptions, nil
	}
	err := json.Unmarshal(*request, &requestOptions)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse options from request, using effective options")
		return &Options{}, err
	}
	return requestOptions, nil
}
