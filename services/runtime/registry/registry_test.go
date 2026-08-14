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
	"strings"
	"testing"

	"github.com/pkg/errors"
)

const arm64Artifact = "example.com/runtime/cdp@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

func TestArtifactForSelectsTheRequestedPlatform(t *testing.T) {
	service := Service{
		Name: "cdp",
		Artifacts: []Artifact{
			{OS: "linux", Architecture: "arm64", Reference: arm64Artifact},
			{OS: "linux", Architecture: "amd64", Reference: cdpArtifact},
		},
	}

	artifact, err := service.ArtifactFor("linux", "amd64")
	if err != nil {
		t.Fatalf("unexpected artifact error: %v", err)
	}
	if artifact != cdpArtifact {
		t.Fatalf("expected the amd64 artifact, got %q", artifact)
	}
}

func TestArtifactForReportsAnUnsupportedPlatform(t *testing.T) {
	service := Service{
		Name:      "cdp",
		Artifacts: []Artifact{{OS: "linux", Architecture: "arm64", Reference: arm64Artifact}},
	}

	_, err := service.ArtifactFor("linux", "amd64")
	if !errors.Is(err, ErrPlatformNotSupported) {
		t.Fatalf("expected an unsupported platform error, got %v", err)
	}
}

func TestArtifactForRequiresAnExplicitArtifactList(t *testing.T) {
	_, err := (Service{Name: "cdp"}).ArtifactFor("linux", "amd64")
	if err == nil || !strings.Contains(err.Error(), "at least one artifact") {
		t.Fatalf("expected a missing artifact list to fail the definition, got %v", err)
	}
}

func TestArtifactForRejectsIncompletePlatforms(t *testing.T) {
	service := Service{
		Name:      "cdp",
		Artifacts: []Artifact{{Architecture: "amd64", Reference: cdpArtifact}},
	}

	if _, err := service.ArtifactFor("linux", "amd64"); err == nil || !strings.Contains(err.Error(), "both os and architecture") {
		t.Fatalf("expected an incomplete platform to fail, got %v", err)
	}
}

func TestArtifactForRejectsDuplicatePlatforms(t *testing.T) {
	service := Service{
		Name: "cdp",
		Artifacts: []Artifact{
			{OS: "linux", Architecture: "amd64", Reference: cdpArtifact},
			{OS: "linux", Architecture: "amd64", Reference: arm64Artifact},
		},
	}

	if _, err := service.ArtifactFor("linux", "amd64"); err == nil || !strings.Contains(err.Error(), "conflicting artifacts") {
		t.Fatalf("expected duplicate platforms to fail, got %v", err)
	}
}

func TestArtifactForRejectsNonCanonicalPlatforms(t *testing.T) {
	service := Service{
		Name:      "cdp",
		Artifacts: []Artifact{{OS: "Linux", Architecture: "AMD64", Reference: cdpArtifact}},
	}

	if _, err := service.ArtifactFor("linux", "amd64"); err == nil || !strings.Contains(err.Error(), "canonical lowercase") {
		t.Fatalf("expected a non-canonical platform to fail, got %v", err)
	}
}

// Every artifact belongs to one registry definition, so a malformed reference
// for another platform makes that definition invalid rather than lying dormant
// until a node of that architecture asks for it.
func TestArtifactForValidatesEveryPublishedReference(t *testing.T) {
	service := Service{
		Name: "cdp",
		Artifacts: []Artifact{
			{OS: "linux", Architecture: "amd64", Reference: cdpArtifact},
			{OS: "linux", Architecture: "arm64", Reference: "example.com/runtime/cdp:latest"},
		},
	}

	if _, err := service.ArtifactFor("linux", "amd64"); err == nil || !strings.Contains(err.Error(), "digest-pinned") {
		t.Fatalf("expected the mutable reference to fail the definition, got %v", err)
	}
}

func TestArtifactForAcceptsOneMultiPlatformIndexForSeveralPlatforms(t *testing.T) {
	service := Service{
		Name: "cdp",
		Artifacts: []Artifact{
			{OS: "linux", Architecture: "amd64", Reference: cdpArtifact},
			{OS: "linux", Architecture: "arm64", Reference: cdpArtifact},
		},
	}

	for _, architecture := range []string{"amd64", "arm64"} {
		artifact, err := service.ArtifactFor("linux", architecture)
		if err != nil {
			t.Fatalf("failed to select the shared index for %s: %v", architecture, err)
		}
		if artifact != cdpArtifact {
			t.Fatalf("selected %q for %s, expected %q", artifact, architecture, cdpArtifact)
		}
	}
}
