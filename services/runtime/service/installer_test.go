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
	"testing"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/services/runtime/registry"
	runtime_lib_service "github.com/mysteriumnetwork/runtime/service"
)

const approvedArtifact = "example.com/runtime/cdp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type installerBackend struct {
	lifecycleBackend

	created   []ApprovedCreateOptions
	deleted   []string
	createErr error
	installed Options
}

func (backend *installerBackend) Create(options ApprovedCreateOptions) error {
	backend.created = append(backend.created, options)
	return backend.createErr
}

func (backend *installerBackend) Delete(name string) error {
	backend.deleted = append(backend.deleted, name)
	return nil
}

func (backend *installerBackend) Get(name string) (ServiceInfo, bool, error) {
	if len(backend.created) == 0 {
		return ServiceInfo{}, false, nil
	}
	options := backend.installed
	options.Name = name
	return ServiceInfo{Name: name, Options: options}, true, nil
}

type fakeRegistry struct {
	entry registry.Service
	err   error
	calls int
}

func (fake *fakeRegistry) ListServices() ([]registry.Service, error) {
	fake.calls++
	if fake.err != nil {
		return nil, fake.err
	}
	if fake.entry.ServiceType() == "" {
		return nil, nil
	}
	return []registry.Service{fake.entry}, nil
}

func approvedEntry() registry.Service {
	entry := registry.Service{
		Name: "cdp",
		Artifacts: []registry.Artifact{{
			OS:           goruntime.GOOS,
			Architecture: goruntime.GOARCH,
			Reference:    approvedArtifact,
		}},
	}
	entry.Manifest.SchemaVersion = 1
	entry.Manifest.Service.Protocol = "tcp"
	entry.Manifest.Service.InternalPort = 9222
	entry.Manifest.Resources = runtime_lib_service.ResourceLimits{
		CPU: "1", Memory: "512MiB", Disk: "512MiB", Pids: 128,
	}
	return entry
}

func installedFrom(entry registry.Service) Options {
	return Options{
		OCIArtifact:    approvedArtifact,
		ServicePort:    entry.Manifest.Service.InternalPort,
		ResourceLimits: entry.Manifest.Resources,
	}
}

func TestInstallUsesTheArtifactTheRegistryPinned(t *testing.T) {
	entry := approvedEntry()
	backend := &installerBackend{installed: installedFrom(entry)}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err != nil {
		t.Fatalf("unexpected install error: %v", err)
	}
	if len(backend.created) != 1 {
		t.Fatalf("expected one create, got %d", len(backend.created))
	}
	created := backend.created[0]
	if created.Name != "runtime-cdp" {
		t.Fatalf("unexpected installed service name %q", created.Name)
	}
	if created.OCIArtifact() != approvedArtifact {
		t.Fatalf("unexpected installed artifact %q", created.OCIArtifact())
	}
	if created.runtimeOptions().OCIArtifact != approvedArtifact {
		t.Fatal("the approved artifact did not reach the runtime backend")
	}
}

func TestInstallRefusesServiceThatIsNotListed(t *testing.T) {
	backend := &installerBackend{}
	installer := NewInstaller(backend, &fakeRegistry{})

	err := installer.Install(CreateOptions{Name: "runtime-miner"})
	if !errors.Is(err, registry.ErrNotListed) {
		t.Fatalf("expected an unlisted service to be refused, got %v", err)
	}
	if len(backend.created) != 0 {
		t.Fatal("an unlisted service reached the runtime backend")
	}
}

// Without a registry there is no approved workload list, so nothing may be
// installed - the node must not fall back to trusting the request.
func TestInstallRefusesEverythingWithoutARegistry(t *testing.T) {
	backend := &installerBackend{}
	installer := NewInstaller(backend, nil)

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err == nil {
		t.Fatal("expected create to be refused without a service registry")
	}
	if len(backend.created) != 0 {
		t.Fatal("a workload was installed without a service registry")
	}
}

// A workload that never installed has nothing to roll back, so a failed
// create must not turn into a delete of whatever else holds that name.
func TestInstallReportsBackendFailureWithoutRollback(t *testing.T) {
	entry := approvedEntry()
	backend := &installerBackend{installed: installedFrom(entry), createErr: errors.New("pull failed")}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err == nil {
		t.Fatal("expected the backend failure to be reported")
	}
	if len(backend.deleted) != 0 {
		t.Fatalf("a failed create must not delete anything, deleted %v", backend.deleted)
	}
}

func TestInstallRemovesServiceThatDoesNotMatchItsRegistryEntry(t *testing.T) {
	entry := approvedEntry()
	installed := installedFrom(entry)
	installed.ServicePort = 8080
	backend := &installerBackend{installed: installed}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	err := installer.Install(CreateOptions{Name: "runtime-cdp"})
	if err == nil {
		t.Fatal("expected a service that contradicts its registry entry to fail")
	}
	if len(backend.deleted) != 1 || backend.deleted[0] != "runtime-cdp" {
		t.Fatalf("expected the mismatched service to be removed, deleted %v", backend.deleted)
	}
}

func TestInstallRejectsResourceLimitsThatContradictTheRegistry(t *testing.T) {
	entry := approvedEntry()
	installed := installedFrom(entry)
	installed.ResourceLimits.Memory = "4GiB"
	backend := &installerBackend{installed: installed}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err == nil {
		t.Fatal("expected contradicting resource limits to fail the install")
	}
}

// Limits are the quantities they denote, not the text they are written in: the
// registry and the artifact may spell one limit differently and still agree.
func TestInstallAcceptsEquivalentlySpelledLimits(t *testing.T) {
	entry := approvedEntry()
	entry.Manifest.Resources = runtime_lib_service.ResourceLimits{
		CPU: "0.50", Memory: "512 MiB", Disk: "0.5GiB", Pids: 128,
	}
	installed := installedFrom(entry)
	installed.ResourceLimits = runtime_lib_service.ResourceLimits{
		CPU: ".5", Memory: "536870912", Disk: "512MiB", Pids: 128,
	}
	backend := &installerBackend{installed: installed}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err != nil {
		t.Fatalf("equivalent limits were treated as a mismatch: %v", err)
	}
}

func TestInstallRejectsUnreadableDeclaredLimit(t *testing.T) {
	entry := approvedEntry()
	entry.Manifest.Resources.Memory = "lots"
	backend := &installerBackend{installed: installedFrom(entry)}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err == nil {
		t.Fatal("expected an unreadable declared limit to fail the install")
	}
}

// Limits the registry leaves out belong to the artifact, which the runtime
// backend defaults and validates on its own.
func TestInstallAcceptsLimitsTheRegistryDoesNotDeclare(t *testing.T) {
	entry := approvedEntry()
	entry.Manifest.Resources = runtime_lib_service.ResourceLimits{}
	backend := &installerBackend{installed: Options{
		ServicePort: entry.Manifest.Service.InternalPort,
		ResourceLimits: runtime_lib_service.ResourceLimits{
			CPU: "1", Memory: "512MiB", Disk: "512MiB", Pids: 128,
		},
	}}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err != nil {
		t.Fatalf("unexpected install error: %v", err)
	}
}

func TestInstallTakesTheStricterIsolationFloor(t *testing.T) {
	tests := []struct {
		requested, published, expected RuntimeLevel
	}{
		{RuntimeLevelUnisolated, RuntimeLevelFull, RuntimeLevelFull},
		{RuntimeLevelFull, RuntimeLevelUnisolated, RuntimeLevelFull},
		{"", RuntimeLevelLimited, RuntimeLevelLimited},
		{RuntimeLevelLimited, "", RuntimeLevelLimited},
	}

	for _, test := range tests {
		entry := approvedEntry()
		entry.MinimumRuntimeLevel = test.published
		backend := &installerBackend{installed: installedFrom(entry)}
		installer := NewInstaller(backend, &fakeRegistry{entry: entry})

		err := installer.Install(CreateOptions{Name: "runtime-cdp", MinimumRuntimeLevel: test.requested})
		if err != nil {
			t.Fatalf("unexpected install error: %v", err)
		}
		if actual := backend.created[0].MinimumRuntimeLevel; actual != test.expected {
			t.Fatalf(
				"requested %q with registry level %q installed as %q, expected %q",
				test.requested, test.published, actual, test.expected,
			)
		}
	}
}

func TestInstallFailsOnAPlatformTheEntryDoesNotCover(t *testing.T) {
	entry := approvedEntry()
	entry.Artifacts = []registry.Artifact{{
		OS:           "plan9",
		Architecture: "sparc",
		Reference:    approvedArtifact,
	}}
	if goruntime.GOOS == "plan9" {
		t.Skip("test platform must not be covered by the entry")
	}

	backend := &installerBackend{installed: installedFrom(entry)}
	installer := NewInstaller(backend, &fakeRegistry{entry: entry})

	if err := installer.Install(CreateOptions{Name: "runtime-cdp"}); err == nil {
		t.Fatal("expected an entry without an artifact for this platform to fail")
	}
	if len(backend.created) != 0 {
		t.Fatal("a workload with no artifact for this platform was installed")
	}
}
