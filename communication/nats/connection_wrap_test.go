/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package nats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitiseServer(t *testing.T) {
	var tests = []struct {
		uri  string
		want string
	}{
		{"127.0.0.1", "nats://127.0.0.1:4222"},
		{"nats://127.0.0.1", "nats://127.0.0.1:4222"},
		{"127.0.0.1:4222", "nats://127.0.0.1:4222"},
		{"nats://127.0.0.1:4222", "nats://127.0.0.1:4222"},

		{"nats://127.0.0.1:4333", "nats://127.0.0.1:4333"},
		{"nats://example.com:4333", "nats://example.com:4333"},
	}

	for _, tc := range tests {
		address, err := SanitiseServer(tc.uri)
		assert.NoError(t, err)
		assert.Equal(t, tc.want, address.String())
	}
}

func TestConnection_Close_NotOpened(t *testing.T) {
	connection := NewConnection("nats://far-server:1234")
	connection.Close()
}

func TestConnection_Close_AfterFailedOpen(t *testing.T) {
	connection := NewConnection("nats://far-server:1234")

	assert.EqualError(t, connection.Open(), "nats: no servers available for connection")
	connection.Close()
}

func TestConnection_Servers(t *testing.T) {
	connection := NewConnection("nats://far-server:1234")
	assert.Equal(t, []string{"nats://far-server:1234"}, connection.Servers())
}
