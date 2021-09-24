/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package domain

import (
	"testing"
)

type testcase struct {
	input  string
	result bool
}

func TestMatch(t *testing.T) {
	l := []string{
		"localhost",
		".localhost",
		".localdomain",
	}
	wl := NewWhitelist(l)

	testVector := []testcase{
		{
			input:  "localhost",
			result: true,
		},
		{
			input:  "localhost1",
			result: false,
		},
		{
			input:  "localhost.localhost",
			result: true,
		},
		{
			input:  "localdomain",
			result: false,
		},
		{
			input:  "localhost.localdomain",
			result: true,
		},
		{
			input:  " localhost ",
			result: true,
		},
		{
			input:  "example.org",
			result: false,
		},
		{
			input:  "localhost.",
			result: true,
		},
		{
			input:  "localhost1.",
			result: false,
		},
		{
			input:  "localhost.localhost.",
			result: true,
		},
		{
			input:  "localdomain.",
			result: false,
		},
		{
			input:  "localhost.localdomain.",
			result: true,
		},
		{
			input:  " localhost. ",
			result: true,
		},
		{
			input:  "example.org.",
			result: false,
		},
	}

	for _, tc := range testVector {
		res := wl.Match(tc.input)
		if res != tc.result {
			t.Errorf("Whitelist(%#v).Match(%#v) returned wrong result. Expected: %v. Got: %v.",
				l, tc.input, tc.result, res)
		}
	}
}

func TestRootWildcard(t *testing.T) {
	l := []string{"."}
	wl := NewWhitelist(l)
	testVector := []testcase{
		{
			input:  "localhost",
			result: true,
		},
		{
			input:  "localhost1",
			result: true,
		},
		{
			input:  "localhost.localhost",
			result: true,
		},
		{
			input:  "localdomain",
			result: true,
		},
		{
			input:  "localhost.localdomain",
			result: true,
		},
		{
			input:  " localhost ",
			result: true,
		},
		{
			input:  "example.org",
			result: true,
		},
		{
			input:  "localhost.",
			result: true,
		},
		{
			input:  "localhost1.",
			result: true,
		},
		{
			input:  "localhost.localhost.",
			result: true,
		},
		{
			input:  "localdomain.",
			result: true,
		},
		{
			input:  "localhost.localdomain.",
			result: true,
		},
		{
			input:  " localhost. ",
			result: true,
		},
		{
			input:  "example.org.",
			result: true,
		},
		{
			input:  ".",
			result: false,
		},
		{
			input:  "",
			result: false,
		},
	}

	for _, tc := range testVector {
		res := wl.Match(tc.input)
		if res != tc.result {
			t.Errorf("Whitelist(%#v).Match(%#v) returned wrong result. Expected: %v. Got: %v.",
				l, tc.input, tc.result, res)
		}
	}
}

func TestRootExact(t *testing.T) {
	l := []string{""}
	wl := NewWhitelist(l)
	testVector := []testcase{
		{
			input:  "localhost",
			result: false,
		},
		{
			input:  "localhost1",
			result: false,
		},
		{
			input:  "localhost.localhost",
			result: false,
		},
		{
			input:  "localdomain",
			result: false,
		},
		{
			input:  "localhost.localdomain",
			result: false,
		},
		{
			input:  " localhost ",
			result: false,
		},
		{
			input:  "example.org",
			result: false,
		},
		{
			input:  "localhost.",
			result: false,
		},
		{
			input:  "localhost1.",
			result: false,
		},
		{
			input:  "localhost.localhost.",
			result: false,
		},
		{
			input:  "localdomain.",
			result: false,
		},
		{
			input:  "localhost.localdomain.",
			result: false,
		},
		{
			input:  " localhost. ",
			result: false,
		},
		{
			input:  "example.org.",
			result: false,
		},
		{
			input:  ".",
			result: true,
		},
		{
			input:  "",
			result: true,
		},
	}

	for _, tc := range testVector {
		res := wl.Match(tc.input)
		if res != tc.result {
			t.Errorf("Whitelist(%#v).Match(%#v) returned wrong result. Expected: %v. Got: %v.",
				l, tc.input, tc.result, res)
		}
	}
}
