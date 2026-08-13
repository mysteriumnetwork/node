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

// Package registry reads the corporate runtime service registry: the list of
// workloads a node is allowed to install. A create request names a service;
// everything executable about it - which artifact to pull and what it promises
// to be - comes from here, never from the request.
package registry

import (
	goruntime "runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	runtime_lib_service "github.com/mysteriumnetwork/runtime/service"
)

// The platform a registry entry has to cover to be installable here. The
// runtime backend pulls the artifact for this same platform.
const (
	hostOS           = goruntime.GOOS
	hostArchitecture = goruntime.GOARCH
)

// Manifest is the workload contract the registry publishes for a service. It
// is the same document the artifact carries as mysterium-runtime.json, which
// is what makes it verifiable after installation.
type Manifest = runtime_lib_service.Manifest

// RuntimeLevel is the isolation floor a workload may demand.
type RuntimeLevel = runtime_lib_service.RuntimeLevel

// ErrNotListed reports that a service is absent from the registry, which is
// the normal answer for a create request the node must refuse.
var ErrNotListed = errors.New("service is not listed in the runtime service registry")

// Artifact is one platform-specific OCI link of a registry entry. Registries
// that publish a single multi-platform index use Service.OCIArtifact instead.
type Artifact struct {
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	OCIArtifact  string `json:"oci_artifact"`
}

// Service is one entry of the registry. Unknown fields are tolerated on
// purpose: the registry is free to publish descriptions, owners or versions
// for humans, and none of it can reach the runtime, because only the fields
// below are ever read.
type Service struct {
	Name string `json:"name"`
	// OCIArtifact is the platform-agnostic link, normally a multi-platform
	// index. Artifacts, when present, takes precedence for a matching platform.
	OCIArtifact string     `json:"oci_artifact,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty"`
	Manifest    Manifest   `json:"manifest"`
	// MinimumRuntimeLevel lets the registry demand a stricter isolation floor
	// than the requester asked for. It can only tighten, never loosen.
	MinimumRuntimeLevel RuntimeLevel `json:"minimum_runtime_level,omitempty"`
}

// ServiceType is the canonical runtime-<name> type this entry installs as.
func (service Service) ServiceType() string {
	return runtime_service.NormalizeServiceType(service.Name)
}

// SelectService returns the single valid definition for serviceName from one
// registry snapshot. Entries with unusable names do not affect other services,
// while duplicate definitions are refused because there is no safe one to
// choose.
func SelectService(services []Service, serviceName string) (Service, error) {
	serviceType := runtime_service.NormalizeServiceType(serviceName)
	if serviceType == "" {
		return Service{}, errors.New("runtime service name is required")
	}

	var selected Service
	matches := 0
	for _, candidate := range services {
		if candidate.ServiceType() != serviceType {
			continue
		}
		selected = candidate
		matches++
	}

	switch matches {
	case 0:
		return Service{}, errors.Wrapf(ErrNotListed, "runtime service %q", serviceType)
	case 1:
		if err := selected.validate(hostOS, hostArchitecture); err != nil {
			return Service{}, err
		}
		return selected, nil
	default:
		return Service{}, errors.Errorf(
			"runtime service registry lists %d conflicting entries for %q", matches, serviceType,
		)
	}
}

// ArtifactFor returns the digest-pinned OCI reference to install on the given
// platform. An entry that lists platform-specific artifacts must cover the
// host platform; falling back to a link built for another platform would
// install a workload that cannot run.
func (service Service) ArtifactFor(os, architecture string) (string, error) {
	var fallback string
	for _, artifact := range service.Artifacts {
		if artifact.OS == "" && artifact.Architecture == "" {
			if fallback == "" {
				fallback = artifact.OCIArtifact
			}
			continue
		}
		if strings.EqualFold(artifact.OS, os) && strings.EqualFold(artifact.Architecture, architecture) {
			return artifact.OCIArtifact, nil
		}
	}
	if fallback == "" {
		fallback = service.OCIArtifact
	}
	if fallback == "" {
		return "", errors.Errorf("registry entry %q has no OCI artifact for %s/%s", service.Name, os, architecture)
	}
	return fallback, nil
}

// validate rejects entries the node must not act on. It runs at lookup rather
// than at decode so that one malformed entry cannot make the rest of the
// registry unusable, and so the operator sees why their own service failed.
func (service Service) validate(os, architecture string) error {
	if service.ServiceType() == "" {
		return errors.Errorf("registry entry has an unusable service name %q", service.Name)
	}

	artifact, err := service.ArtifactFor(os, architecture)
	if err != nil {
		return err
	}
	if _, err := name.NewDigest(artifact, name.StrictValidation); err != nil {
		return errors.Wrapf(err, "registry entry %q must pin its OCI artifact by digest", service.Name)
	}

	return service.validateManifest()
}

// validateManifest checks the parts of the published contract the node relies
// on. The artifact's own copy is validated in full by the runtime backend; this
// only makes sure the registry is describing a service of the shape a node can
// expose at all.
//
// The manifest schema version is not checked here: the registry API version in
// the endpoint path carries it, so a node that can read this listing at all is
// reading the schema it was built for.
func (service Service) validateManifest() error {
	manifest := service.Manifest
	if !strings.EqualFold(manifest.Service.Protocol, "tcp") {
		return errors.Errorf("registry entry %q must declare a tcp service", service.Name)
	}
	if manifest.Service.InternalPort < 1 || manifest.Service.InternalPort > 65535 {
		return errors.Errorf(
			"registry entry %q declares an out of range internal_port %d",
			service.Name, manifest.Service.InternalPort,
		)
	}
	return nil
}
