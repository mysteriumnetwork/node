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
	goruntime "runtime"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/runtime/registry"
)

// Registry lists the workloads this node is permitted to install.
type Registry interface {
	// Lookup returns the approved definition published for a service name, or
	// an error wrapping registry.ErrNotListed when there is none.
	Lookup(serviceName string) (registry.Service, error)
}

// Installer turns a create request into an installed workload. It is the only
// way to reach Backend.Create, because only this package can build the
// ApprovedCreateOptions that Create demands.
type Installer interface {
	Install(options CreateOptions) error
}

// ErrNotConfigured reports that this node cannot install workloads at all,
// because it has no backend or no service registry. It is distinct from a
// refused install: nothing about the request would make it succeed, so callers
// that report failures onwards should say so rather than blame the request.
var ErrNotConfigured = errors.New("runtime services are not configured")

// NewInstaller returns an Installer that admits only workloads the registry
// publishes. A nil registry is a configuration error rather than an open door:
// every create is refused until one is configured.
func NewInstaller(backend Backend, serviceRegistry Registry) Installer {
	return &registryInstaller{backend: backend, registry: serviceRegistry}
}

type registryInstaller struct {
	backend  Backend
	registry Registry
}

// Install resolves the requested name against the registry, installs exactly
// the artifact the registry pinned for it, and then checks that what got
// installed matches what the registry promised.
func (installer *registryInstaller) Install(options CreateOptions) error {
	if installer.backend == nil {
		return errors.Wrap(ErrNotConfigured, "runtime backend is not configured")
	}
	if installer.registry == nil {
		return errors.Wrap(ErrNotConfigured, "runtime service registry is not configured; runtime services cannot be created")
	}
	if options.Name == "" {
		return errors.New("runtime service name is required")
	}

	entry, err := installer.registry.Lookup(options.Name)
	if err != nil {
		return err
	}
	artifact, err := entry.ArtifactFor(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return err
	}

	approved := ApprovedCreateOptions{
		CreateOptions: CreateOptions{
			Name: options.Name,
			// The registry can raise the isolation floor for its own workloads
			// but must never be able to lower one the requester asked for.
			MinimumRuntimeLevel: strictestRuntimeLevel(options.MinimumRuntimeLevel, entry.MinimumRuntimeLevel),
		},
		ociArtifact: artifact,
	}

	if err := installer.backend.Create(approved); err != nil {
		return err
	}

	if err := installer.verify(options.Name, entry); err != nil {
		// The definition is installed but is not the one the registry
		// described, so it is removed rather than left for a later start.
		if deleteErr := installer.backend.Delete(options.Name); deleteErr != nil {
			log.Warn().Err(deleteErr).Str("service", options.Name).
				Msg("Failed to remove runtime service that did not match its registry entry")
		}
		return err
	}
	return nil
}

// verify compares the installed definition against the manifest the registry
// published. The artifact is digest-pinned, so a difference does not mean the
// image changed underneath us: it means the registry entry no longer describes
// what it points at, and the node refuses to publish a service on that basis.
func (installer *registryInstaller) verify(serviceName string, entry registry.Service) error {
	installed, exists, err := installer.backend.Get(serviceName)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("runtime service %q was not installed", serviceName)
	}

	if port := entry.Manifest.Service.InternalPort; port != installed.Options.ServicePort {
		return errors.Errorf(
			"runtime service %q listens on port %d but the service registry declares %d",
			serviceName, installed.Options.ServicePort, port,
		)
	}

	declared := entry.Manifest.Resources
	actual := installed.Options.ResourceLimits
	for _, limit := range []struct {
		name             string
		declared, actual string
		parse            func(string) (float64, bool)
	}{
		{"cpu", declared.CPU, actual.CPU, parseCPUCores},
		{"memory", declared.Memory, actual.Memory, parseByteSize},
		{"disk", declared.Disk, actual.Disk, parseByteSize},
	} {
		// An omitted limit is left to the artifact, which the runtime backend
		// defaults and validates on its own.
		if limit.declared == "" {
			continue
		}
		// Limits are compared as the quantities they denote, so a difference in
		// spelling between the registry and the artifact is not mistaken for a
		// difference in what the workload is allowed to use.
		declaredValue, ok := limit.parse(limit.declared)
		if !ok {
			return errors.Errorf(
				"the service registry declares an unreadable %s limit %q for runtime service %q",
				limit.name, limit.declared, serviceName,
			)
		}
		actualValue, ok := limit.parse(limit.actual)
		if !ok {
			return errors.Errorf(
				"runtime service %q was installed with an unreadable %s limit %q",
				serviceName, limit.name, limit.actual,
			)
		}
		if declaredValue != actualValue {
			return errors.Errorf(
				"runtime service %q runs with %s limit %q but the service registry declares %q",
				serviceName, limit.name, limit.actual, limit.declared,
			)
		}
	}
	if declared.Pids != 0 && declared.Pids != actual.Pids {
		return errors.Errorf(
			"runtime service %q runs with pids limit %d but the service registry declares %d",
			serviceName, actual.Pids, declared.Pids,
		)
	}

	return nil
}

// runtimeLevelRank orders isolation floors from weakest to strictest. An
// unknown or empty level ranks lowest, so it never wins over a stated one.
func runtimeLevelRank(level RuntimeLevel) int {
	switch level {
	case RuntimeLevelUnisolated:
		return 1
	case RuntimeLevelLimited:
		return 2
	case RuntimeLevelFull:
		return 3
	default:
		return 0
	}
}

func strictestRuntimeLevel(requested, published RuntimeLevel) RuntimeLevel {
	if runtimeLevelRank(published) > runtimeLevelRank(requested) {
		return published
	}
	return requested
}
