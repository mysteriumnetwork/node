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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	core_service "github.com/mysteriumnetwork/node/core/service"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	"github.com/mysteriumnetwork/node/services/runtime/registry"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

type reconcileBackend struct {
	installed []ServiceInfo
	listErr   error
	deleted   []string
	deleteErr error
}

func (backend *reconcileBackend) Create(ApprovedCreateOptions) error { return nil }
func (backend *reconcileBackend) Delete(name string) error {
	if backend.deleteErr != nil {
		return backend.deleteErr
	}
	backend.deleted = append(backend.deleted, name)
	remaining := make([]ServiceInfo, 0, len(backend.installed))
	for _, info := range backend.installed {
		if info.Name != name {
			remaining = append(remaining, info)
		}
	}
	backend.installed = remaining
	return nil
}
func (backend *reconcileBackend) Start(string) error                         { return nil }
func (backend *reconcileBackend) Stop(string) error                          { return nil }
func (backend *reconcileBackend) SetDesiredState(string, ServiceState) error { return nil }
func (backend *reconcileBackend) Get(string) (ServiceInfo, bool, error) {
	return ServiceInfo{}, false, nil
}
func (backend *reconcileBackend) DialTCP(string, int) (net.Conn, error) { return nil, nil }
func (backend *reconcileBackend) List() ([]ServiceInfo, error) {
	if backend.listErr != nil {
		return nil, backend.listErr
	}
	return backend.installed, nil
}
func (backend *reconcileBackend) Status() RuntimeStatus {
	return RuntimeStatus{Level: RuntimeLevelFull}
}
func (backend *reconcileBackend) Availability() Availability { return Availability{Available: true} }
func (backend *reconcileBackend) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return runtime_capabilities.RuntimeCapabilities{}, runtime_capabilities.DetailedCapabilities{}
}

type listingRegistry struct {
	mu       sync.Mutex
	services []registry.Service
	err      error
	calls    int
}

func (fake *listingRegistry) ListServices() ([]registry.Service, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.calls++
	if fake.err != nil {
		return nil, fake.err
	}
	return append([]registry.Service(nil), fake.services...), nil
}

func (fake *listingRegistry) publish(serviceTypes ...string) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.services = make([]registry.Service, 0, len(serviceTypes))
	for _, serviceType := range serviceTypes {
		entry := approvedEntry()
		entry.Name = serviceType
		fake.services = append(fake.services, entry)
	}
	fake.err = nil
}

func (fake *listingRegistry) fail(err error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.err = err
}

func (fake *listingRegistry) syncs() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.calls
}

type fakeInstances struct {
	running []*core_service.Instance
	stopped []core_service.ID
	stopErr error
}

func (instances *fakeInstances) List(bool) []*core_service.Instance { return instances.running }
func (instances *fakeInstances) Stop(id core_service.ID) error {
	if instances.stopErr != nil {
		return instances.stopErr
	}
	instances.stopped = append(instances.stopped, id)
	remaining := make([]*core_service.Instance, 0, len(instances.running))
	for _, instance := range instances.running {
		if instance.ID != id {
			remaining = append(remaining, instance)
		}
	}
	instances.running = remaining
	return nil
}

func newTestReconciler(backend Backend, serviceRegistry Registry, instances InstanceLifecycle) *RegistryReconciler {
	reconciler := NewRegistryReconciler(backend, serviceRegistry, instances)
	reconciler.interval = time.Millisecond
	return reconciler
}

// The count is the whole point: one sync that omits a service is not evidence
// enough to take a working workload off a node.
func TestUnlistedServiceIsRemovedOnlyAfterThresholdSyncs(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: "runtime-cdp"}}}
	instances := &fakeInstances{running: []*core_service.Instance{
		{ID: "instance-1", Type: "runtime-cdp"},
	}}
	serviceRegistry := &listingRegistry{}
	serviceRegistry.publish("runtime-miner")
	reconciler := newTestReconciler(backend, serviceRegistry, instances)

	for sync := 1; sync < DefaultReconcileMissThreshold; sync++ {
		reconciler.Reconcile()
		if len(backend.deleted) != 0 {
			t.Fatalf("runtime service removed after %d syncs, threshold is %d", sync, DefaultReconcileMissThreshold)
		}
	}

	reconciler.Reconcile()
	if len(backend.deleted) != 1 || backend.deleted[0] != "runtime-cdp" {
		t.Fatalf("deleted %v, expected runtime-cdp", backend.deleted)
	}
	// A definition removed underneath a running instance would leave a
	// published proposal for a workload that no longer exists.
	if len(instances.stopped) != 1 || instances.stopped[0] != "instance-1" {
		t.Fatalf("stopped %v, expected the running runtime-cdp instance", instances.stopped)
	}
}

// The condition that protects a node from its own registry: an outage must not
// look like a removal, no matter how long it lasts.
func TestUnavailableRegistryNeverRemovesServices(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: "runtime-cdp"}}}
	serviceRegistry := &listingRegistry{err: errors.New("runtime service registry returned status 500")}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold*3; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v while the registry was unavailable", backend.deleted)
	}
}

// A failed sync must not even count towards a removal, or an outage spread
// across two good syncs would still age a listed service out.
func TestFailedSyncDoesNotCountTowardsRemoval(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: "runtime-cdp"}}}
	serviceRegistry := &listingRegistry{}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold-1; i++ {
		reconciler.Reconcile()
	}
	serviceRegistry.fail(errors.New("failed to reach runtime service registry"))
	for i := 0; i < DefaultReconcileMissThreshold; i++ {
		reconciler.Reconcile()
	}
	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v, a failed sync counted towards removal", backend.deleted)
	}

	serviceRegistry.fail(nil)
	reconciler.Reconcile()
	if len(backend.deleted) != 1 {
		t.Fatalf("deleted %v, expected the removal to resume after the outage", backend.deleted)
	}
}

// Only consecutive absences remove a service: an entry that comes back starts
// over, so a registry republishing an entry cannot remove it on the next miss.
func TestReappearingServiceResetsTheCount(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{
		Name:    "runtime-cdp",
		Options: Options{OCIArtifact: approvedArtifact},
	}}}
	serviceRegistry := &listingRegistry{}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold-1; i++ {
		reconciler.Reconcile()
	}
	serviceRegistry.publish("runtime-cdp")
	reconciler.Reconcile()

	serviceRegistry.publish()
	for i := 0; i < DefaultReconcileMissThreshold-1; i++ {
		reconciler.Reconcile()
	}
	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v, the count was not reset by the service reappearing", backend.deleted)
	}

	reconciler.Reconcile()
	if len(backend.deleted) != 1 {
		t.Fatalf("deleted %v, expected removal after a full run of absences", backend.deleted)
	}
}

// A listed service is never touched, however many syncs run.
func TestListedServicesAreKept(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{
		{Name: "runtime-cdp", Options: Options{OCIArtifact: approvedArtifact}},
		{Name: "runtime-miner", Options: Options{OCIArtifact: approvedArtifact}},
	}}
	serviceRegistry := &listingRegistry{}
	serviceRegistry.publish("runtime-cdp", "runtime-miner")
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold*2; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v, expected listed services to be kept", backend.deleted)
	}
}

func TestServiceWithWithdrawnArtifactIsRemovedAfterThresholdSyncs(t *testing.T) {
	const replacementArtifact = "example.com/runtime/cdp@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	backend := &reconcileBackend{installed: []ServiceInfo{{
		Name:    "runtime-cdp",
		Options: Options{OCIArtifact: approvedArtifact},
	}}}
	entry := approvedEntry()
	entry.Artifacts[0].Reference = replacementArtifact
	serviceRegistry := &listingRegistry{services: []registry.Service{entry}}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 1 || backend.deleted[0] != "runtime-cdp" {
		t.Fatalf("deleted %v, expected the no-longer-approved artifact to be removed", backend.deleted)
	}
}

func TestServiceWithdrawnFromTheHostPlatformIsRemovedAfterThresholdSyncs(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{
		Name:    "runtime-cdp",
		Options: Options{OCIArtifact: approvedArtifact},
	}}}
	entry := approvedEntry()
	entry.Artifacts = []registry.Artifact{{
		OS:           "unsupported-test-os",
		Architecture: "unsupported-test-architecture",
		Reference:    approvedArtifact,
	}}
	serviceRegistry := &listingRegistry{services: []registry.Service{entry}}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 1 || backend.deleted[0] != "runtime-cdp" {
		t.Fatalf("deleted %v, expected the unsupported platform workload to be removed", backend.deleted)
	}
}

func TestMalformedPlatformDefinitionDoesNotAgeServiceOut(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{
		Name:    "runtime-cdp",
		Options: Options{OCIArtifact: approvedArtifact},
	}}}
	entry := approvedEntry()
	entry.Artifacts = append(entry.Artifacts, entry.Artifacts[0])
	serviceRegistry := &listingRegistry{services: []registry.Service{entry}}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold*2; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v while the platform definition was malformed", backend.deleted)
	}
}

func TestAmbiguousRegistryDefinitionDoesNotAgeServiceOut(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{
		Name:    "runtime-cdp",
		Options: Options{OCIArtifact: approvedArtifact},
	}}}
	entry := approvedEntry()
	serviceRegistry := &listingRegistry{services: []registry.Service{entry, entry}}
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold*2; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v while the registry definition was ambiguous", backend.deleted)
	}
}

// The parent runtime service is node configuration, not a workload the registry
// publishes, so it must never be reconciled away.
func TestParentRuntimeServiceIsNeverRemoved(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: runtime_service.ServiceType}}}
	reconciler := newTestReconciler(backend, &listingRegistry{}, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold*2; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v, the parent runtime service must be left alone", backend.deleted)
	}
}

// A removal that fails is retried on the next sync rather than granting the
// service another full run of absences.
func TestFailedRemovalIsRetriedOnTheNextSync(t *testing.T) {
	backend := &reconcileBackend{
		installed: []ServiceInfo{{Name: "runtime-cdp"}},
		deleteErr: errors.New("backend is busy"),
	}
	reconciler := newTestReconciler(backend, &listingRegistry{}, &fakeInstances{})

	for i := 0; i < DefaultReconcileMissThreshold; i++ {
		reconciler.Reconcile()
	}
	if len(backend.deleted) != 0 {
		t.Fatal("delete reported success while the backend was failing")
	}

	backend.deleteErr = nil
	reconciler.Reconcile()
	if len(backend.deleted) != 1 {
		t.Fatalf("deleted %v, expected the removal to be retried immediately", backend.deleted)
	}
}

// A definition must outlive a failed stop: removing it would leave the instance
// serving a workload nothing can reinstall.
func TestServiceIsKeptWhenItsInstanceCannotBeStopped(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: "runtime-cdp"}}}
	instances := &fakeInstances{
		running: []*core_service.Instance{{ID: "instance-1", Type: "runtime-cdp"}},
		stopErr: errors.New("stop failed"),
	}
	reconciler := newTestReconciler(backend, &listingRegistry{}, instances)

	for i := 0; i < DefaultReconcileMissThreshold; i++ {
		reconciler.Reconcile()
	}

	if len(backend.deleted) != 0 {
		t.Fatalf("deleted %v while its instance was still running", backend.deleted)
	}
}

// Reconciliation lives exactly as long as the parent runtime service.
func TestRunStopsWithTheParentService(t *testing.T) {
	backend := &reconcileBackend{installed: []ServiceInfo{{Name: "runtime-cdp"}}}
	serviceRegistry := &listingRegistry{}
	serviceRegistry.publish("runtime-cdp")
	reconciler := newTestReconciler(backend, serviceRegistry, &fakeInstances{})

	stopped := make(chan struct{})
	done := make(chan struct{})
	go func() {
		reconciler.Run(stopped)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if serviceRegistry.syncs() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("reconciliation never ran")
		case <-time.After(time.Millisecond):
		}
	}

	close(stopped)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reconciliation kept running after the parent service stopped")
	}
}
