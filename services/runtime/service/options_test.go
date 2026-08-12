package service

import (
	"encoding/json"
	"reflect"
	"strings"
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
	raw := json.RawMessage(`{"name":"cdp","rootfs":"/host"}`)
	if _, err := ParseJSONCreateOptions(raw); err == nil {
		t.Fatal("expected rootfs override to be rejected")
	}
}

func TestParseJSONCreateOptionsRejectsCallerChosenArtifact(t *testing.T) {
	raw := json.RawMessage(`{"name":"cdp","oci_artifact":"example.invalid/x@sha256:abc"}`)
	_, err := ParseJSONCreateOptions(raw)
	if err == nil {
		t.Fatal("expected a caller-supplied OCI artifact to be rejected")
	}
	if !strings.Contains(err.Error(), "service registry") {
		t.Fatalf("expected the error to point at the service registry, got %v", err)
	}
}

func TestParseJSONCreateOptionsAcceptsNameOnlyRequest(t *testing.T) {
	raw := json.RawMessage(`{
		"name":"cdp",
		"minimum_runtime_level":"unisolated"
	}`)
	options, err := ParseJSONCreateOptions(raw)
	if err != nil {
		t.Fatalf("unexpected create options error: %v", err)
	}
	if options.Name != "cdp" {
		t.Fatalf("expected service name cdp, got %q", options.Name)
	}
	if options.MinimumRuntimeLevel != RuntimeLevelUnisolated {
		t.Fatalf("expected unisolated minimum runtime level, got %q", options.MinimumRuntimeLevel)
	}

	approved := ApprovedCreateOptions{CreateOptions: options, ociArtifact: "example.invalid/x@sha256:abc"}
	if approved.runtimeOptions().MinimumRuntimeLevel != runtime_service.RuntimeLevelUnisolated {
		t.Fatal("minimum runtime level was not forwarded to the runtime")
	}
	if approved.runtimeOptions().OCIArtifact != "example.invalid/x@sha256:abc" {
		t.Fatal("approved artifact was not forwarded to the runtime")
	}
}

// A create that names the service through the service type carries no options
// at all, and must not be treated as a malformed request.
func TestParseJSONCreateOptionsAcceptsAbsentOptions(t *testing.T) {
	if _, err := ParseJSONCreateOptions(nil); err != nil {
		t.Fatalf("expected absent create options to be accepted: %v", err)
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
