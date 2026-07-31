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

package control

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	runtime_service_options "github.com/mysteriumnetwork/node/services/runtime/service"
	"github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func TestResolveRuntimeServiceType_RuntimeBaseUsesName(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime", "CDP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime-cdp" {
		t.Fatalf("expected runtime-cdp, got %q", actual)
	}
}

func TestResolveRuntimeServiceType_RuntimePrefixedTypePassesThrough(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime-cdp", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime-cdp" {
		t.Fatalf("expected runtime-cdp, got %q", actual)
	}
}

func TestResolveRuntimeServiceType_LegacyDottedTypeFails(t *testing.T) {
	if _, err := resolveRuntimeServiceType("runtime.cdp", ""); err == nil {
		t.Fatal("expected legacy dotted runtime service type to be rejected")
	}
}

func TestResolveRuntimeServiceType_RuntimeServiceNameNormalized(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime", " CDP Browser ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime-cdp-browser" {
		t.Fatalf("expected runtime-cdp-browser, got %q", actual)
	}
}

func TestResolveRuntimeServiceType_RuntimeBaseWithoutNameFails(t *testing.T) {
	_, err := resolveRuntimeServiceType("runtime", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateRuntimeCommand_AllowsDerivedRuntimeWhenRuntimeActive(t *testing.T) {
	err := validateRuntimeCommand([]contract.ServiceInfoDTO{{Type: "runtime"}}, controlMessageItem{
		Command: "start",
		Service: "runtime-cdp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeCommand_RejectsDerivedRuntimeWhenRuntimeInactive(t *testing.T) {
	err := validateRuntimeCommand(nil, controlMessageItem{
		Command: "start",
		Service: "runtime-cdp",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateRuntimeCommand_AllowsStartingBaseRuntimeWithoutRuntimeActive(t *testing.T) {
	err := validateRuntimeCommand(nil, controlMessageItem{
		Command: "start",
		Service: "runtime",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeCommand_CreateRequiresRuntimeActive(t *testing.T) {
	err := validateRuntimeCommand(nil, controlMessageItem{
		Command: "create",
		Service: "runtime",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToRuntimeServiceOptions_MapsOnlyCreateContract(t *testing.T) {
	options := toRuntimeServiceOptions("runtime-cdp", RuntimeServiceOptions{
		OCIArtifact:         "example.com/runtime/cdp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinimumRuntimeLevel: runtime_service_options.RuntimeLevelUnisolated,
	})

	expected := runtime_service_options.CreateOptions{
		Name:                "runtime-cdp",
		OCIArtifact:         "example.com/runtime/cdp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinimumRuntimeLevel: runtime_service_options.RuntimeLevelUnisolated,
	}

	if options.Name != expected.Name ||
		options.OCIArtifact != expected.OCIArtifact ||
		options.MinimumRuntimeLevel != expected.MinimumRuntimeLevel {
		t.Fatalf("runtime options were not mapped completely: %#v", options)
	}
}

func TestParseRuntimeCreateOptions_RejectsHostControlledExecution(t *testing.T) {
	_, err := (&ControlPlane{}).parseRuntimeCreateOptions(controlMessageItem{
		Options: json.RawMessage(`{
			"oci_artifact": "example.com/runtime/cdp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"exec": ["/bin/sh"]
		}`),
	})
	if err == nil {
		t.Fatal("expected host-controlled exec to be rejected")
	}
}

func TestRejectRuntimeCommandOptions(t *testing.T) {
	if err := rejectRuntimeCommandOptions(controlMessageItem{
		Options: json.RawMessage(`{"exec":["/bin/sh"]}`),
	}); err == nil {
		t.Fatal("expected runtime start options to be rejected")
	}
	if err := rejectRuntimeCommandOptions(controlMessageItem{
		Options: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("expected empty runtime start options to be accepted: %v", err)
	}
}

func TestRequiresRuntimeAvailabilityAllowsCleanup(t *testing.T) {
	for _, command := range []string{"stop", "delete"} {
		if requiresRuntimeAvailability(controlMessageItem{
			Service: "runtime-workload",
			Command: command,
		}) {
			t.Fatalf("%s must remain available for runtime cleanup", command)
		}
	}
	for _, command := range []string{"create", "start", "restart"} {
		if !requiresRuntimeAvailability(controlMessageItem{
			Service: "runtime-workload",
			Command: command,
		}) {
			t.Fatalf("%s must check the current runtime level", command)
		}
	}
}

func TestHandler_RestartPreservesMasterStopThenStartBehavior(t *testing.T) {
	var requests []string
	var startRequest contract.ServiceStartRequest

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.Path)
		response.Header().Set("Content-Type", "application/json")

		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/services":
			_ = json.NewEncoder(response).Encode(contract.ServiceListResponse{{
				ID:         "service-id",
				ProviderID: "provider-id",
				Type:       "wireguard",
			}})
		case request.Method == http.MethodDelete && request.URL.Path == "/services/service-id":
			response.WriteHeader(http.StatusAccepted)
		case request.Method == http.MethodPost && request.URL.Path == "/services":
			if err := json.NewDecoder(request.Body).Decode(&startRequest); err != nil {
				t.Errorf("failed to decode start request: %v", err)
			}
			_ = json.NewEncoder(response).Encode(contract.ServiceInfoDTO{})
		default:
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	host, portValue, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatalf("failed to split test server address: %v", err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("failed to parse test server port: %v", err)
	}

	controlPlane := &ControlPlane{
		api:      client.NewClient(host, port),
		identity: "provider-id",
	}
	err = controlPlane.handler(controlMessage{{
		Service: "wireguard",
		Command: "restart",
	}})
	if err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}

	expectedRequests := []string{
		"GET /services",
		"DELETE /services/service-id",
		"POST /services",
	}
	if len(requests) != len(expectedRequests) {
		t.Fatalf("expected requests %v, got %v", expectedRequests, requests)
	}
	for i := range expectedRequests {
		if requests[i] != expectedRequests[i] {
			t.Fatalf("expected requests %v, got %v", expectedRequests, requests)
		}
	}
	if startRequest.ProviderID != "provider-id" || startRequest.Type != "wireguard" {
		t.Fatalf("unexpected restart start request: %#v", startRequest)
	}
}
