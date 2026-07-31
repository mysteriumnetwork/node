package service

import (
	"net"
	"sync"
	"testing"
	"time"

	core_service "github.com/mysteriumnetwork/node/core/service"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

type lifecycleBackend struct {
	startEntered chan struct{}
	releaseStart chan struct{}
	stopCalled   chan string
	startOnce    sync.Once
	status       RuntimeStatus
}

func (backend *lifecycleBackend) Create(CreateOptions) error { return nil }
func (backend *lifecycleBackend) Delete(string) error        { return nil }
func (backend *lifecycleBackend) Start(name string) error {
	backend.startOnce.Do(func() { close(backend.startEntered) })
	<-backend.releaseStart
	return nil
}
func (backend *lifecycleBackend) Stop(name string) error {
	backend.stopCalled <- name
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
	}
	manager := NewManager(backend, "runtime-test", nil)
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
