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
	"fmt"

	"github.com/mysteriumnetwork/node/core/service"
	runtime_service "github.com/mysteriumnetwork/runtime/service"
)

// Options describe the OCI artifact and runtime metadata needed to launch a runtime-backed service.
type Options struct {
	Name           string            `json:"name,omitempty"`
	OCIArtifact    string            `json:"oci_artifact,omitempty"`
	RootFS         string            `json:"rootfs,omitempty"`
	Exec           string            `json:"exec,omitempty"`
	ServicePort    int               `json:"service_port,omitempty"`
	ResourceLimits ResourceLimits    `json:"resource_limits,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
}

// ResourceLimits describe runtime resource constraints for spawned workloads.
type ResourceLimits = runtime_service.ResourceLimits

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
	if err != nil {
		return requestOptions, err
	}

	if requestOptions.ServicePort < 0 || requestOptions.ServicePort > 65535 {
		return requestOptions, fmt.Errorf("service_port must be between 0 and 65535")
	}

	return requestOptions, nil
}

func (options Options) runtimeOptions() runtime_service.Options {
	return runtime_service.Options{
		Name:        options.Name,
		OCIArtifact: options.OCIArtifact,
		RootFS:      options.RootFS,
		Exec:        options.Exec,
		ResourceLimits: runtime_service.ResourceLimits{
			CPU:    options.ResourceLimits.CPU,
			Memory: options.ResourceLimits.Memory,
			Disk:   options.ResourceLimits.Disk,
		},
		Env: options.Env,
	}
}

func optionsFromRuntime(options runtime_service.Options) Options {
	return Options{
		Name:        options.Name,
		OCIArtifact: options.OCIArtifact,
		RootFS:      options.RootFS,
		Exec:        options.Exec,
		ResourceLimits: ResourceLimits{
			CPU:    options.ResourceLimits.CPU,
			Memory: options.ResourceLimits.Memory,
			Disk:   options.ResourceLimits.Disk,
		},
		Env: options.Env,
	}
}

func mergeOptions(base, override Options) Options {
	result := base
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.OCIArtifact != "" {
		result.OCIArtifact = override.OCIArtifact
	}
	if override.RootFS != "" {
		result.RootFS = override.RootFS
	}
	if override.Exec != "" {
		result.Exec = override.Exec
	}
	if override.ServicePort > 0 {
		result.ServicePort = override.ServicePort
	}
	if override.ResourceLimits.CPU != "" {
		result.ResourceLimits.CPU = override.ResourceLimits.CPU
	}
	if override.ResourceLimits.Memory != "" {
		result.ResourceLimits.Memory = override.ResourceLimits.Memory
	}
	if override.ResourceLimits.Disk != "" {
		result.ResourceLimits.Disk = override.ResourceLimits.Disk
	}
	if len(override.Env) > 0 {
		result.Env = override.Env
	}
	return result
}
