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

func TestParseByteSize(t *testing.T) {
	tests := map[string]float64{
		"128MiB":    128 * 1024 * 1024,
		"128 mib":   128 * 1024 * 1024,
		" 0.5GiB ":  512 * 1024 * 1024,
		"536870912": 512 * 1024 * 1024,
		"512MB":     512 * 1000 * 1000,
		"1024":      1024,
		"2048B":     2048,
	}
	for value, expected := range tests {
		actual, ok := parseByteSize(value)
		if !ok {
			t.Fatalf("failed to read size %q", value)
		}
		if actual != expected {
			t.Fatalf("size %q read as %v, expected %v", value, actual, expected)
		}
	}

	for _, value := range []string{"", "   ", "lots", "-1MiB", "MiB", "1.2.3GiB"} {
		if _, ok := parseByteSize(value); ok {
			t.Fatalf("expected size %q to be unreadable", value)
		}
	}
}

func TestParseCPUCores(t *testing.T) {
	tests := map[string]float64{
		"0.5":  0.5,
		".5":   0.5,
		"0.50": 0.5,
		" 1 ":  1,
		"2":    2,
	}
	for value, expected := range tests {
		actual, ok := parseCPUCores(value)
		if !ok {
			t.Fatalf("failed to read CPU limit %q", value)
		}
		if actual != expected {
			t.Fatalf("CPU limit %q read as %v, expected %v", value, actual, expected)
		}
	}

	for _, value := range []string{"", "half", "-1"} {
		if _, ok := parseCPUCores(value); ok {
			t.Fatalf("expected CPU limit %q to be unreadable", value)
		}
	}
}
