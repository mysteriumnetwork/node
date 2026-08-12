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

package contract

import (
	runtime_service "github.com/mysteriumnetwork/node/services/runtime/service"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

// RuntimeServiceInstallRequest requests installation of a runtime workload.
// The artifact is never named here: it is resolved from the service registry
// by name, so a caller cannot install a workload the registry does not publish.
// swagger:model RuntimeServiceInstallRequestDTO
type RuntimeServiceInstallRequest struct {
	// Name of the service as the service registry publishes it. It is
	// normalized to the canonical runtime-<name> service type, so both "cdp"
	// and "runtime-cdp" name the same service.
	// required: true
	// example: cdp
	Name string `json:"name"`

	// Lowest isolation level this workload may run at. The service registry can
	// raise this floor for its own entries but never lower it.
	// required: false
	// example: limited
	MinimumRuntimeLevel runtime_service.RuntimeLevel `json:"minimum_runtime_level,omitempty"`
}

// RuntimeServiceInfoDTO describes a runtime workload definition installed on
// the node.
// swagger:model RuntimeServiceInfoDTO
type RuntimeServiceInfoDTO struct {
	// Canonical runtime-<name> service type. This is the value to pass as
	// "type" when starting the service through POST /services.
	// example: runtime-cdp
	Name string `json:"name"`

	// What the workload is doing right now.
	// example: active
	State runtime_service.ServiceState `json:"state"`

	// Intent recorded by the runtime backend. It survives a node restart and
	// drives what gets started again on boot, so it can differ from state
	// while a service is being restored.
	// example: active
	DesiredState runtime_service.ServiceState `json:"desired_state"`

	// ID of the running node-side service instance, empty when the service is
	// not running. Use it to stop the service through DELETE /services/:id.
	// example: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
	ServiceID string `json:"service_id,omitempty"`

	// Immutable definition the runtime backend resolved from the artifact
	// manifest.
	Options runtime_service.Options `json:"options"`
}

// RuntimeServiceListResponse is a list of installed runtime workload
// definitions, running or not.
// swagger:model RuntimeServiceListResponse
type RuntimeServiceListResponse struct {
	Services []RuntimeServiceInfoDTO `json:"services"`
}

// RuntimeStatusDTO reports whether this host can run workloads at all, and how
// completely it can isolate them.
// swagger:model RuntimeStatusDTO
type RuntimeStatusDTO struct {
	// Whether workloads can be run on this host.
	// example: true
	Available bool `json:"available"`

	// Why workloads cannot be run, empty when they can.
	// example: user namespaces are unavailable
	Reason string `json:"reason,omitempty"`

	// Isolation level, profiles and blocking reasons as probed on this host.
	Runtime runtime_service.RuntimeStatus `json:"runtime"`

	// Host capabilities the runtime depends on.
	Capabilities runtime_capabilities.RuntimeCapabilities `json:"capabilities"`

	// Per-capability probe results behind the summary above.
	DetailedCapabilities runtime_capabilities.DetailedCapabilities `json:"detailed_capabilities"`
}
