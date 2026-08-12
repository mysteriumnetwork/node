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

import "testing"

func TestArtifactForPrefersTheHostPlatform(t *testing.T) {
	service := Service{
		Name:        "cdp",
		OCIArtifact: "example.com/runtime/cdp@sha256:index",
		Artifacts: []Artifact{
			{OS: "linux", Architecture: "arm64", OCIArtifact: "example.com/runtime/cdp@sha256:arm64"},
			{OS: "Linux", Architecture: "AMD64", OCIArtifact: "example.com/runtime/cdp@sha256:amd64"},
		},
	}

	artifact, err := service.ArtifactFor("linux", "amd64")
	if err != nil {
		t.Fatalf("unexpected artifact error: %v", err)
	}
	if artifact != "example.com/runtime/cdp@sha256:amd64" {
		t.Fatalf("expected the amd64 artifact, got %q", artifact)
	}
}

// An entry that lists platforms but not this one has nothing installable here;
// falling back to another platform's link would install an unrunnable workload.
func TestArtifactForFailsOnUncoveredPlatform(t *testing.T) {
	service := Service{
		Name:      "cdp",
		Artifacts: []Artifact{{OS: "linux", Architecture: "arm64", OCIArtifact: "example.com/runtime/cdp@sha256:arm64"}},
	}

	if _, err := service.ArtifactFor("linux", "amd64"); err == nil {
		t.Fatal("expected an uncovered platform to fail")
	}
}

func TestArtifactForFallsBackToPlatformAgnosticLink(t *testing.T) {
	service := Service{
		Name:        "cdp",
		OCIArtifact: "example.com/runtime/cdp@sha256:index",
	}

	artifact, err := service.ArtifactFor("linux", "amd64")
	if err != nil {
		t.Fatalf("unexpected artifact error: %v", err)
	}
	if artifact != "example.com/runtime/cdp@sha256:index" {
		t.Fatalf("expected the platform-agnostic artifact, got %q", artifact)
	}
}
