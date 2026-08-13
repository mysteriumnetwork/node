package service

import (
	"net"
	"sync"
	"testing"
	"time"

	core_service "github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

type lifecycleBackend struct {
	startEntered chan struct{}
	releaseStart chan struct{}
	stopCalled   chan string
	desiredSet   chan ServiceState
	startOnce    sync.Once
	status       RuntimeStatus
}

func (backend *lifecycleBackend) Create(ApprovedCreateOptions) error { return nil }
func (backend *lifecycleBackend) Delete(string) error                { return nil }
func (backend *lifecycleBackend) Start(name string) error {
	backend.startOnce.Do(func() { close(backend.startEntered) })
	<-backend.releaseStart
	return nil
}
func (backend *lifecycleBackend) Stop(name string) error {
	backend.stopCalled <- name
	return nil
}
func (backend *lifecycleBackend) SetDesiredState(name string, state ServiceState) error {
	backend.desiredSet <- state
	return nil
}
func (backend *lifecycleBackend) Get(string) (ServiceInfo, bool, error) {
	return ServiceInfo{}, false, nil
}
func (backend *lifecycleBackend) DialTCP(string, int) (net.Conn, error) { return nil, nil }
func (backend *lifecycleBackend) List() ([]ServiceInfo, error)          { return nil, nil }
func (backend *lifecycleBackend) Status() RuntimeStatus {
	if backend.status.Level != "" {
		return backend.status
	}
	return RuntimeStatus{Level: RuntimeLevelFull}
}
func (backend *lifecycleBackend) Availability() Availability {
	return Availability{Available: true}
}
func (backend *lifecycleBackend) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return runtime_capabilities.RuntimeCapabilities{}, runtime_capabilities.DetailedCapabilities{}
}

func TestUnavailableReasonUsesRuntimeStatus(t *testing.T) {
	backend := &lifecycleBackend{
		status: RuntimeStatus{
			Level:           RuntimeLevelUnavailable,
			BlockingReasons: []string{"missing user namespace support", "missing runc"},
		},
	}

	reason, unavailable := UnavailableReason(backend)
	if !unavailable || reason != "missing user namespace support, missing runc" {
		t.Fatalf("unexpected availability result: unavailable=%v reason=%q", unavailable, reason)
	}
}

func TestUnavailableReasonAllowsDegradedRuntimeLevels(t *testing.T) {
	for _, level := range []RuntimeLevel{RuntimeLevelUnisolated, RuntimeLevelLimited, RuntimeLevelFull} {
		backend := &lifecycleBackend{status: RuntimeStatus{Level: level}}
		if reason, unavailable := UnavailableReason(backend); unavailable {
			t.Fatalf("runtime level %q was rejected: %s", level, reason)
		}
	}
}

func TestManagerStopDuringStartCleansUpStartedWorkload(t *testing.T) {
	backend := &lifecycleBackend{
		startEntered: make(chan struct{}),
		releaseStart: make(chan struct{}),
		stopCalled:   make(chan string, 1),
		desiredSet:   make(chan ServiceState, 1),
	}
	manager := NewManager(backend, "runtime-test", nil, nil, nil)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- manager.Serve(&core_service.Instance{Type: "runtime-test"})
	}()

	select {
	case <-backend.startEntered:
	case <-time.After(time.Second):
		t.Fatal("backend start was not called")
	}
	if err := manager.Stop(); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	close(backend.releaseStart)

	select {
	case name := <-backend.stopCalled:
		if name != "runtime-test" {
			t.Fatalf("stopped %q, expected runtime-test", name)
		}
	case <-time.After(time.Second):
		t.Fatal("workload was orphaned after Stop raced with Start")
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected serve error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not return")
	}
}

// The parent runtime service is the trigger for restoring runtime-* services,
// and it must hand over the identity the workloads are published under.
func TestParentRuntimeServiceReconcilesOnce(t *testing.T) {
	reconciled := make(chan identity.Identity, 2)
	manager := NewManager(&lifecycleBackend{}, "", nil, func(providerID identity.Identity, stopped <-chan struct{}) {
		reconciled <- providerID
	}, nil)

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- manager.Serve(&core_service.Instance{
			Type:       runtime_service.ServiceType,
			ProviderID: identity.FromAddress("0xprovider"),
		})
	}()

	select {
	case providerID := <-reconciled:
		if providerID.Address != "0xprovider" {
			t.Fatalf("reconciled with %q, want the parent service provider", providerID.Address)
		}
	case <-time.After(time.Second):
		t.Fatal("parent runtime service did not reconcile runtime workloads")
	}

	if err := manager.Stop(); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected serve error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("parent runtime service did not stop")
	}

	if len(reconciled) != 0 {
		t.Fatal("runtime workloads were reconciled more than once")
	}
}

// A runtime-* service starts its own workload; only the parent triggers a pass.
func TestRuntimeWorkloadServiceDoesNotReconcile(t *testing.T) {
	reconciled := make(chan identity.Identity, 1)
	backend := &lifecycleBackend{
		startEntered: make(chan struct{}),
		releaseStart: make(chan struct{}),
		stopCalled:   make(chan string, 1),
		desiredSet:   make(chan ServiceState, 1),
	}
	close(backend.releaseStart)
	manager := NewManager(backend, "", nil, func(providerID identity.Identity, stopped <-chan struct{}) {
		reconciled <- providerID
	}, nil)

	go manager.Serve(&core_service.Instance{Type: "runtime-foo"})

	select {
	case <-backend.startEntered:
	case <-time.After(time.Second):
		t.Fatal("runtime workload was not started")
	}
	_ = manager.Stop()

	if len(reconciled) != 0 {
		t.Fatal("a runtime workload service triggered reconciliation")
	}
}

// The bug this guards: a node shutdown stops every service through the same
// call an operator stop uses. Recording passive there erased the desired state
// on every restart, so a service that was active never came back.
func TestNodeShutdownKeepsServiceRecordedActive(t *testing.T) {
	backend := &lifecycleBackend{
		startEntered: make(chan struct{}),
		releaseStart: make(chan struct{}),
		stopCalled:   make(chan string, 1),
		desiredSet:   make(chan ServiceState, 1),
	}
	close(backend.releaseStart)
	shuttingDown := true
	manager := NewManager(backend, "runtime-cdp", nil, nil, func() bool { return shuttingDown })

	go manager.Serve(&core_service.Instance{Type: "runtime-cdp"})
	select {
	case <-backend.startEntered:
	case <-time.After(time.Second):
		t.Fatal("workload was not started")
	}

	if err := manager.Stop(); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	select {
	case <-backend.stopCalled:
	case <-time.After(time.Second):
		t.Fatal("workload was not stopped on shutdown")
	}
	if len(backend.desiredSet) != 0 {
		t.Fatal("node shutdown overwrote the desired state; the service would not come back")
	}
}

// An operator stopping a service must survive the next restart.
func TestOperatorStopRecordsServicePassive(t *testing.T) {
	backend := &lifecycleBackend{
		startEntered: make(chan struct{}),
		releaseStart: make(chan struct{}),
		stopCalled:   make(chan string, 1),
		desiredSet:   make(chan ServiceState, 1),
	}
	close(backend.releaseStart)
	manager := NewManager(backend, "runtime-cdp", nil, nil, func() bool { return false })

	go manager.Serve(&core_service.Instance{Type: "runtime-cdp"})
	select {
	case <-backend.startEntered:
	case <-time.After(time.Second):
		t.Fatal("workload was not started")
	}

	if err := manager.Stop(); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	select {
	case state := <-backend.desiredSet:
		if state != ServiceStatePassive {
			t.Fatalf("recorded %q, want %q", state, ServiceStatePassive)
		}
	case <-time.After(time.Second):
		t.Fatal("operator stop did not record the service passive")
	}
}
