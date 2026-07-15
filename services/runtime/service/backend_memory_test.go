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

package service

import "testing"

func TestMemoryBackend_CreateStartStopDeleteLifecycle(t *testing.T) {
	backend := NewMemoryBackend()
	options := Options{Name: "runtime.cdp", RootFS: "oci://example/cdp", Command: "serve"}

	if err := backend.Create(options); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	services, err := backend.List()
	if err != nil {
		t.Fatalf("list after create failed: %v", err)
	}
	if len(services) != 1 || services[0].State != ServiceStatePassive {
		t.Fatalf("expected one passive service after create, got %+v", services)
	}

	if err := backend.Start(nil, options); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	services, err = backend.List()
	if err != nil {
		t.Fatalf("list after start failed: %v", err)
	}
	if len(services) != 1 || services[0].State != ServiceStateActive {
		t.Fatalf("expected one active service after start, got %+v", services)
	}

	if err := backend.Stop(options); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	services, err = backend.List()
	if err != nil {
		t.Fatalf("list after stop failed: %v", err)
	}
	if len(services) != 1 || services[0].State != ServiceStatePassive {
		t.Fatalf("expected one passive service after stop, got %+v", services)
	}

	if err := backend.Delete(options.Name); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	services, err = backend.List()
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(services) != 0 {
		t.Fatalf("expected no services after delete, got %+v", services)
	}
}
