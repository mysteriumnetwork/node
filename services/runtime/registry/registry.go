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

var (
	// ErrNotListed reports that a service is absent from the registry, which is
	// the normal answer for a create request the node must refuse.
	ErrNotListed = errors.New("service is not listed in the runtime service registry")
	// ErrPlatformNotSupported reports that a listed service does not publish an
	// artifact for this node's platform. Reconciliation treats this as an
	// intentional withdrawal from that platform, rather than malformed data.
	ErrPlatformNotSupported = errors.New("platform is not supported by the runtime service registry")
)

// Artifact is one platform-specific, immutable OCI reference. A multi-platform
// index can be listed for several platforms by repeating its digest-pinned
// reference in each corresponding entry.
type Artifact struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Reference    string `json:"reference"`
}

// Service is one entry of the registry. Unknown fields are tolerated on
// purpose: the registry is free to publish descriptions, owners or versions
// for humans, and none of it can reach the runtime, because only the fields
// below are ever read.
type Service struct {
	Name      string     `json:"name"`
	Artifacts []Artifact `json:"artifacts"`
	Manifest  Manifest   `json:"manifest"`
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

// ArtifactFor returns the single digest-pinned OCI reference the service
// publishes for the given platform. The registry contract is deliberately
// strict: every entry must describe a complete, unique platform and every
// reference must be immutable, even when it targets another platform.
func (service Service) ArtifactFor(os, architecture string) (string, error) {
	if len(service.Artifacts) == 0 {
		return "", errors.Errorf("registry entry %q must publish at least one artifact", service.Name)
	}

	seen := make(map[string]struct{}, len(service.Artifacts))
	selected := ""
	for index, artifact := range service.Artifacts {
		artifactOS := strings.TrimSpace(artifact.OS)
		artifactArchitecture := strings.TrimSpace(artifact.Architecture)
		if artifactOS == "" || artifactArchitecture == "" {
			return "", errors.Errorf(
				"registry entry %q artifact %d must declare both os and architecture",
				service.Name, index,
			)
		}
		if artifact.OS != strings.ToLower(artifactOS) ||
			artifact.Architecture != strings.ToLower(artifactArchitecture) {
			return "", errors.Errorf(
				"registry entry %q artifact %d must use canonical lowercase os and architecture",
				service.Name, index,
			)
		}

		platform := artifact.OS + "/" + artifact.Architecture
		if _, exists := seen[platform]; exists {
			return "", errors.Errorf(
				"registry entry %q lists conflicting artifacts for %s",
				service.Name, platform,
			)
		}
		seen[platform] = struct{}{}

		if _, err := name.NewDigest(artifact.Reference, name.StrictValidation); err != nil {
			return "", errors.Wrapf(
				err,
				"registry entry %q artifact for %s must use a digest-pinned reference",
				service.Name, platform,
			)
		}
		if artifact.OS == os && artifact.Architecture == architecture {
			selected = artifact.Reference
		}
	}

	if selected == "" {
		return "", errors.Wrapf(
			ErrPlatformNotSupported,
			"registry entry %q publishes no artifact for %s/%s",
			service.Name, os, architecture,
		)
	}
	return selected, nil
}

// validate rejects entries the node must not act on. It runs at lookup rather
// than at decode so that one malformed entry cannot make the rest of the
// registry unusable, and so the operator sees why their own service failed.
func (service Service) validate(os, architecture string) error {
	if service.ServiceType() == "" {
		return errors.Errorf("registry entry has an unusable service name %q", service.Name)
	}

	if _, err := service.ArtifactFor(os, architecture); err != nil {
		return err
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
