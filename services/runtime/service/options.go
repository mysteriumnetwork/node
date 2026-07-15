/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/core/service"
)

// Options describe the OCI artifact and runtime metadata needed to launch a runtime-backed service.
type Options struct {
	Name           string            `json:"name,omitempty"`
	RootFS         string            `json:"rootfs,omitempty"`
	Command        string            `json:"command,omitempty"`
	ResourceLimits ResourceLimits    `json:"resource_limits,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
}

// ResourceLimits describe runtime resource constraints for spawned workloads.
type ResourceLimits struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	Disk   string `json:"disk,omitempty"`
}

// GetOptions returns empty runtime options by default.
func GetOptions() Options {
	return Options{}
}

// ParseJSONOptions parses runtime service options from JSON.
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	requestOptions := GetOptions()
	if request == nil {
		return requestOptions, nil
	}

	err := json.Unmarshal(*request, &requestOptions)
	return requestOptions, err
}
