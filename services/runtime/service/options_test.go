package service

import (
	"encoding/json"
	"reflect"
	"testing"

	runtime_service "github.com/mysteriumnetwork/runtime/service"
)

func TestParseJSONStartOptionsRejectsDefinitionOverrides(t *testing.T) {
	raw := json.RawMessage(`{"service_port":8080}`)
	if _, err := ParseJSONStartOptions(&raw); err == nil {
		t.Fatal("expected runtime start override to be rejected")
	}
}

func TestParseJSONCreateOptionsRejectsUnknownFields(t *testing.T) {
	raw := json.RawMessage(`{"oci_artifact":"example.invalid/x@sha256:abc","rootfs":"/host"}`)
	if _, err := ParseJSONCreateOptions(raw); err == nil {
		t.Fatal("expected rootfs override to be rejected")
	}
}

func TestParseJSONCreateOptionsAcceptsMinimumRuntimeLevel(t *testing.T) {
	raw := json.RawMessage(`{
		"oci_artifact":"example.invalid/x@sha256:abc",
		"minimum_runtime_level":"unisolated"
	}`)
	options, err := ParseJSONCreateOptions(raw)
	if err != nil {
		t.Fatalf("unexpected create options error: %v", err)
	}
	if options.MinimumRuntimeLevel != RuntimeLevelUnisolated {
		t.Fatalf("expected unisolated minimum runtime level, got %q", options.MinimumRuntimeLevel)
	}
	if options.runtimeOptions().MinimumRuntimeLevel != runtime_service.RuntimeLevelUnisolated {
		t.Fatal("minimum runtime level was not forwarded to the runtime")
	}
}

func TestOptionsFromRuntimePreservesIsolationMetadata(t *testing.T) {
	runtimeOptions := runtime_service.Options{
		Name:        "runtime-test",
		OCIArtifact: "example.invalid/x@sha256:abc",
		ServicePort: 8080,
		Process: runtime_service.ProcessDefinition{
			Args: []string{"/workload"},
			Cwd:  "/",
			UID:  1000,
			GID:  1000,
		},
		ResourceLimits: runtime_service.ResourceLimits{
			CPU:    "1",
			Memory: "512MiB",
			Disk:   "512MiB",
			Pids:   128,
		},
		Isolation: runtime_service.IsolationProfile{
			Name:  runtime_service.BestEffortIsolationProfile,
			Level: runtime_service.RuntimeLevelLimited,
			Features: runtime_service.IsolationFeatures{
				MountNamespaces: true,
				Seccomp:         true,
			},
		},
		MinimumRuntimeLevel: runtime_service.RuntimeLevelLimited,
	}

	actual := optionsFromRuntime(runtimeOptions)
	if !reflect.DeepEqual(actual.Process, runtimeOptions.Process) {
		t.Fatalf("process metadata was not preserved: %#v", actual.Process)
	}
	if !reflect.DeepEqual(actual.ResourceLimits, runtimeOptions.ResourceLimits) {
		t.Fatalf("resource limits were not preserved: %#v", actual.ResourceLimits)
	}
	if !reflect.DeepEqual(actual.Isolation, runtimeOptions.Isolation) {
		t.Fatalf("isolation profile was not preserved: %#v", actual.Isolation)
	}
	if actual.MinimumRuntimeLevel != RuntimeLevelLimited {
		t.Fatalf("minimum runtime level was not preserved: %q", actual.MinimumRuntimeLevel)
	}
}
