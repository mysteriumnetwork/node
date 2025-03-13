/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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
	"fmt"

	"github.com/mysteriumnetwork/node/core/service"
)

// Options describes options which are required to start Quic service.
type Options struct{}

// DefaultOptions is a quic service configuration that will be used if no options provided.
var DefaultOptions = Options{}

// GetOptions returns effective Quic service options from application configuration.
func GetOptions() Options {
	return Options{}
}

// ParseJSONOptions function fills in Quic options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	requestOptions := GetOptions()
	if request == nil {
		return requestOptions, nil
	}

	opts := DefaultOptions
	err := json.Unmarshal(*request, &opts)

	return opts, fmt.Errorf("failed to parse quic options: %w", err)
}

// MarshalJSON implements json.Marshaler interface to provide human readable configuration.
func (o Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct{}{})
}

// UnmarshalJSON implements json.Unmarshaler interface to receive human readable configuration.
func (o *Options) UnmarshalJSON(data []byte) error {
	var options struct{}

	if err := json.Unmarshal(data, &options); err != nil {
		return fmt.Errorf("failed to unmarshal quic options: %w", err)
	}

	return nil
}
