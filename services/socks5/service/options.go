/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
)

// Options describes options which are required to start SOCKS5 service.
type Options struct {
	Port port.Port
}

// DefaultOptions is a SOCKS5 service configuration that will be used if no options provided.
var DefaultOptions = Options{
	Port: port.Port(8080),
}

// GetOptions returns effective SOCKS5 service options from application configuration.
func GetOptions() Options {
	return Options{
		Port: port.Port(config.GetInt(config.FlagSOCKS5Port)),
	}
}

// ParseJSONOptions function fills in SOCKS5 options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	var requestOptions = GetOptions()
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
		Port int `json:"port"`
	}{
		Port: o.Port.Num(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (o *Options) UnmarshalJSON(data []byte) error {
	var options struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(data, &options); err != nil {
		return err
	}

	if options.Port != 0 {
		o.Port = port.Port(options.Port)
	}

	return nil
}
