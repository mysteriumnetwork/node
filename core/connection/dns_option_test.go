/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDNSOption(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		input     string
		expect    DNSOption
		expectErr bool
	}{
		{input: "auto", expect: DNSOptionAuto},
		{input: "provider", expect: DNSOptionProvider},
		{input: "system", expect: DNSOptionSystem},
		{input: "1.1.1.1,9.9.9.9", expect: DNSOption("1.1.1.1,9.9.9.9")},
		{input: "1.1.1.1", expect: DNSOption("1.1.1.1")},
		{input: "", expect: DNSOption("")},
		{input: "AA", expectErr: true},
		{input: "512.512.512.512", expectErr: true},
		{input: "1.1.1.1,512.512.512.512", expectErr: true},
	}
	for i, tt := range tests {
		option, err := NewDNSOption(tt.input)
		assert.Equal(tt.expectErr, err != nil, "%v: expected err = %v, actual err = %v", i, tt.expectErr, err)
		assert.Equal(tt.expect, option)
	}
}

func TestDNSOption_UnmarshalJSON(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		input        string
		expectOption DNSOption
		expectErr    bool
	}{
		{input: "\"auto\"", expectOption: DNSOptionAuto},
		{input: "\"provider\"", expectOption: DNSOptionProvider},
		{input: "\"system\"", expectOption: DNSOptionSystem},
		{input: "\"1.1.1.1,9.9.9.9\"", expectOption: DNSOption("1.1.1.1,9.9.9.9")},
		{input: "\"9.9.9.9\"", expectOption: DNSOption("9.9.9.9")},
		{input: "\"\"", expectOption: DNSOption("")},
		{input: "", expectErr: true},
		{input: "\"AA\"", expectErr: true},
		{input: "\"512.512.512.512\"", expectErr: true},
		{input: "\"1.1.1.1,512.512.512.512\"", expectErr: true},
	}
	for i, tt := range tests {
		var option DNSOption
		err := option.UnmarshalJSON([]byte(tt.input))
		assert.Equal(tt.expectErr, err != nil, "%v: expected err = %v, actual err = %v", i, tt.expectErr, err)
		assert.Equal(tt.expectOption, option)
	}
}

func TestDNSOption_Exact(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		option        DNSOption
		expectServers []string
		expectOK      bool
	}{
		{option: DNSOptionAuto, expectOK: false},
		{option: DNSOptionProvider, expectOK: false},
		{option: DNSOptionSystem, expectOK: false},
		{option: DNSOption("1.1.1.1,9.9.9.9"), expectServers: []string{"1.1.1.1", "9.9.9.9"}, expectOK: true},
		{option: DNSOption("9.9.9.9"), expectServers: []string{"9.9.9.9"}, expectOK: true},
		{option: DNSOption(""), expectServers: nil, expectOK: true},
	}
	for _, tt := range tests {
		servers, ok := tt.option.Exact()
		assert.Equal(tt.expectOK, ok)
		assert.Equal(tt.expectServers, servers)
	}
}
