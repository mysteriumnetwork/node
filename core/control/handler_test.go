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
	"testing"

	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func TestResolveRuntimeServiceType_RuntimeBaseUsesName(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime", "CDP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime.cdp" {
		t.Fatalf("expected runtime.cdp, got %q", actual)
	}
}

func TestResolveRuntimeServiceType_RuntimePrefixedTypePassesThrough(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime.cdp", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime.cdp" {
		t.Fatalf("expected runtime.cdp, got %q", actual)
	}
}

func TestResolveRuntimeServiceType_RuntimeServiceNameNormalized(t *testing.T) {
	actual, err := resolveRuntimeServiceType("runtime", " CDP Browser ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual != "runtime.cdp-browser" {
		t.Fatalf("expected runtime.cdp-browser, got %q", actual)
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
		Service: "runtime.cdp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeCommand_RejectsDerivedRuntimeWhenRuntimeInactive(t *testing.T) {
	err := validateRuntimeCommand(nil, controlMessageItem{
		Command: "start",
		Service: "runtime.cdp",
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
