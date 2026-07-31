package service

import (
	"testing"

	runtime_service "github.com/mysteriumnetwork/node/services/runtime/service"
)

type runtimeBackendStub struct {
	runtime_service.Backend
	status runtime_service.RuntimeStatus
}

func (backend runtimeBackendStub) Status() runtime_service.RuntimeStatus {
	return backend.status
}

// Availability deliberately disagrees with Status to ensure proposal gating
// follows the scheduler-facing runtime level.
func (runtimeBackendStub) Availability() runtime_service.Availability {
	return runtime_service.Availability{Available: true}
}

func TestShouldSkipRuntimeServiceWhenBackendUnavailable(t *testing.T) {
	skip, reason := shouldSkipRuntimeService("runtime", nil)
	if !skip || reason == "" {
		t.Fatalf("expected unavailable runtime to be skipped, got skip=%v reason=%q", skip, reason)
	}
	skip, _ = shouldSkipRuntimeService("wireguard", nil)
	if skip {
		t.Fatal("runtime availability must not affect existing services")
	}
}

func TestShouldSkipRuntimeServiceWhenStatusIsUnavailable(t *testing.T) {
	backend := runtimeBackendStub{status: runtime_service.RuntimeStatus{
		Level:           runtime_service.RuntimeLevelUnavailable,
		BlockingReasons: []string{"runtime is unsupported on this host"},
	}}

	for _, serviceType := range []string{"runtime", "runtime-cdp"} {
		skip, reason := shouldSkipRuntimeService(serviceType, backend)
		if !skip || reason != "runtime is unsupported on this host" {
			t.Fatalf("expected %q to be skipped, got skip=%v reason=%q", serviceType, skip, reason)
		}
	}
}
