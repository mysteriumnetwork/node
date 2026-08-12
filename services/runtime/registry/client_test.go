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

package registry

import (
	"net/http"
	"net/http/httptest"
	goruntime "runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
)

const cdpArtifact = "example.com/runtime/cdp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// listing wraps entries in the envelope the registry serves them in.
func listing(entries string) string {
	return `{"services": [` + entries + `]}`
}

func cdpEntry(name string) string {
	return `{
		"name": "` + name + `",
		"description": "ignored by the node",
		"oci_artifact": "` + cdpArtifact + `",
		"manifest": {
			"service": {"protocol": "tcp", "internal_port": 9222},
			"resources": {"cpu": "1", "memory": "512MiB", "disk": "512MiB", "pids": 128}
		}
	}`
}

func testClient(t *testing.T, body string) (*Client, *int64) {
	t.Helper()

	var requests int64
	// TLS, because the client refuses to read a listing it cannot authenticate.
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		atomic.AddInt64(&requests, 1)
		if request.URL.Path != servicesPath {
			t.Errorf("unexpected registry path %q", request.URL.Path)
			response.WriteHeader(http.StatusNotFound)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return clientFor(t, server), &requests
}

// clientFor builds a client the way production does and then teaches it to
// trust the test server's certificate.
func clientFor(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("failed to create registry client: %v", err)
	}
	client.http = server.Client()
	client.http.Timeout = requestTimeout
	return client
}

func TestLookupResolvesServiceByAnyFormOfItsName(t *testing.T) {
	client, _ := testClient(t, listing(cdpEntry("cdp")))

	for _, requested := range []string{"cdp", "CDP", " CDP ", "runtime-cdp"} {
		service, err := client.Lookup(requested)
		if err != nil {
			t.Fatalf("lookup of %q failed: %v", requested, err)
		}
		if service.ServiceType() != "runtime-cdp" {
			t.Fatalf("lookup of %q resolved to %q", requested, service.ServiceType())
		}
		artifact, err := service.ArtifactFor(goruntime.GOOS, goruntime.GOARCH)
		if err != nil {
			t.Fatalf("lookup of %q returned no artifact: %v", requested, err)
		}
		if artifact != cdpArtifact {
			t.Fatalf("lookup of %q returned artifact %q", requested, artifact)
		}
	}
}

// Pins the shape the registry actually serves: entries wrapped under
// "services", with the manifest schema version carried by the API version
// rather than repeated in each entry.
func TestLookupReadsTheDeployedListing(t *testing.T) {
	client, _ := testClient(t, `{
		"services": [
			{
				"name": "cdp",
				"description": "Lightpanda-based CDP test service",
				"oci_artifact": "`+cdpArtifact+`",
				"minimum_runtime_level": "unisolated",
				"manifest": {
					"service": {"protocol": "tcp", "internal_port": 9222},
					"resources": {"cpu": "0.5", "memory": "128MiB", "disk": "256MiB", "pids": 16}
				}
			}
		]
	}`)

	service, err := client.Lookup("cdp")
	if err != nil {
		t.Fatalf("lookup of a wrapped listing failed: %v", err)
	}
	if service.ServiceType() != "runtime-cdp" {
		t.Fatalf("unexpected service type %q", service.ServiceType())
	}
	if service.MinimumRuntimeLevel != RuntimeLevel("unisolated") {
		t.Fatalf("unexpected minimum runtime level %q", service.MinimumRuntimeLevel)
	}
	if service.Manifest.Resources.Memory != "128MiB" || service.Manifest.Resources.Pids != 16 {
		t.Fatalf("published resource limits were not read: %#v", service.Manifest.Resources)
	}
}

func TestLookupRefusesServiceThatIsNotListed(t *testing.T) {
	client, requests := testClient(t, listing(cdpEntry("cdp")))

	_, err := client.Lookup("miner")
	if !errors.Is(err, ErrNotListed) {
		t.Fatalf("expected an unlisted service to be refused, got %v", err)
	}
	// The listing was just read, so it is authoritative and nothing is re-read.
	if atomic.LoadInt64(requests) != 1 {
		t.Fatalf("expected a single registry request, saw %d", atomic.LoadInt64(requests))
	}
}

// A name missing from a cached listing may simply be newer than the cache, so
// a service is only refused after the registry itself has been asked again.
func TestLookupRevalidatesAMissAgainstTheRegistry(t *testing.T) {
	client, requests := testClient(t, listing(cdpEntry("cdp")))

	if _, err := client.Lookup("cdp"); err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if _, err := client.Lookup("miner"); !errors.Is(err, ErrNotListed) {
		t.Fatalf("expected an unlisted service to be refused, got %v", err)
	}
	if atomic.LoadInt64(requests) != 2 {
		t.Fatalf("expected the cached miss to be revalidated, saw %d requests", atomic.LoadInt64(requests))
	}
}

func TestLookupServesRepeatedHitsFromCache(t *testing.T) {
	client, requests := testClient(t, listing(cdpEntry("cdp")))

	for i := 0; i < 3; i++ {
		if _, err := client.Lookup("cdp"); err != nil {
			t.Fatalf("lookup failed: %v", err)
		}
	}
	if atomic.LoadInt64(requests) != 1 {
		t.Fatalf("expected one registry request, saw %d", atomic.LoadInt64(requests))
	}
}

func TestLookupRejectsMutableArtifactReference(t *testing.T) {
	client, _ := testClient(t, listing(`{
		"name": "cdp",
		"oci_artifact": "example.com/runtime/cdp:latest",
		"manifest": {"service": {"protocol": "tcp", "internal_port": 9222}}
	}`))

	_, err := client.Lookup("cdp")
	if err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("expected a tag-based artifact to be rejected, got %v", err)
	}
}

func TestLookupRejectsUnusableManifest(t *testing.T) {
	tests := map[string]string{
		"non tcp service":   `{"service": {"protocol": "udp", "internal_port": 9222}}`,
		"port out of range": `{"service": {"protocol": "tcp", "internal_port": 0}}`,
	}

	for description, manifest := range tests {
		client, _ := testClient(t, listing(`{
			"name": "cdp",
			"oci_artifact": "`+cdpArtifact+`",
			"manifest": `+manifest+`
		}`))

		if _, err := client.Lookup("cdp"); err == nil {
			t.Fatalf("expected %s to be rejected", description)
		}
	}
}

func TestLookupRefusesAmbiguousEntries(t *testing.T) {
	client, _ := testClient(t, listing(cdpEntry("cdp")+","+cdpEntry("CDP")))

	_, err := client.Lookup("cdp")
	if err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("expected conflicting entries to be refused, got %v", err)
	}
}

// One unusable entry must not make the rest of the registry unreachable.
func TestLookupIgnoresEntriesWithoutUsableNames(t *testing.T) {
	client, _ := testClient(t, listing(`{"name": "---"},`+cdpEntry("cdp")))

	if _, err := client.Lookup("cdp"); err != nil {
		t.Fatalf("lookup failed alongside an unusable entry: %v", err)
	}
}

func TestLookupReportsRegistryFailures(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := clientFor(t, server)
	if _, err := client.Lookup("cdp"); err == nil {
		t.Fatal("expected a failing registry to fail the lookup")
	} else if errors.Is(err, ErrNotListed) {
		t.Fatalf("a failing registry must not read as an unlisted service: %v", err)
	}
}

func TestServicesEndpoint(t *testing.T) {
	tests := map[string]string{
		"https://registry.example.com":          "https://registry.example.com/v1/service",
		"https://registry.example.com/":         "https://registry.example.com/v1/service",
		"https://registry.example.com/registry": "https://registry.example.com/registry/v1/service",
	}
	for address, expected := range tests {
		actual, err := servicesEndpoint(address)
		if err != nil {
			t.Fatalf("failed to build endpoint for %q: %v", address, err)
		}
		if actual != expected {
			t.Fatalf("endpoint for %q was %q, expected %q", address, actual, expected)
		}
	}

	for _, address := range []string{"", "   ", "file:///etc/services", "registry.example.com", "https://"} {
		if _, err := servicesEndpoint(address); err == nil {
			t.Fatalf("expected registry address %q to be rejected", address)
		}
	}
}

// The listing decides which workloads this node runs, so a registry that can be
// rewritten in flight is refused outright: pinned digests are no help when the
// same party supplies them.
func TestServicesEndpointRefusesUnauthenticatedRegistry(t *testing.T) {
	for _, address := range []string{
		"http://registry.example.com",
		"http://registry.example.com:8080/registry",
		"http://10.0.0.5:8080",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
	} {
		if _, err := servicesEndpoint(address); err == nil {
			t.Fatalf("expected plaintext registry address %q to be rejected", address)
		}
	}
}
