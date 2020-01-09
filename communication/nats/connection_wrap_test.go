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
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseServerURI(t *testing.T) {
	var tests = []struct {
		uri         string
		wantAddress *url.URL
		wantError   error
	}{
		{"127.0.0.1", &url.URL{Scheme: "nats", Host: "127.0.0.1:4222"}, nil},
		{"nats://127.0.0.1", &url.URL{Scheme: "nats", Host: "127.0.0.1:4222"}, nil},
		{"127.0.0.1:4222", &url.URL{Scheme: "nats", Host: "127.0.0.1:4222"}, nil},
		{"nats://127.0.0.1:4222", &url.URL{Scheme: "nats", Host: "127.0.0.1:4222"}, nil},

		{"nats://127.0.0.1:4333", &url.URL{Scheme: "nats", Host: "127.0.0.1:4333"}, nil},
		{"nats://example.com:4333", &url.URL{Scheme: "nats", Host: "example.com:4333"}, nil},

		{"nats:// example.com", nil, errors.New(`failed to parse NATS server URI "nats:// example.com": parse nats:// example.com: invalid character " " in host name`)},
		{"nats://example.com:a", nil, errors.New(`failed to parse NATS server URI "nats://example.com:a": parse nats://example.com:a: invalid port ":a" after host`)},
	}

	for _, tc := range tests {
		address, err := ParseServerURI(tc.uri)
		if tc.wantError != nil {
			assert.EqualError(t, err, tc.wantError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tc.wantAddress, address)
	}
}

func TestConnectionWrap_NewConnection(t *testing.T) {
	connection, err := newConnection("nats://127.0.0.1:4222")
	assert.NoError(t, err)
	assert.Nil(t, connection.Conn)
	assert.Equal(t, []string{"nats://127.0.0.1:4222"}, connection.servers)

	connection, err = newConnection("nats://127.0.0.1")
	assert.NoError(t, err)
	assert.Nil(t, connection.Conn)
	assert.Equal(t, []string{"nats://127.0.0.1:4222"}, connection.servers)

	connection, err = newConnection("nats:// example.com")
	assert.EqualError(t, err, `failed to parse NATS server URI "nats:// example.com": parse nats:// example.com: invalid character " " in host name`)
	assert.Nil(t, connection)
}

func TestConnectionWrap_Close_AfterFailedOpen(t *testing.T) {
	connection, _ := newConnection("nats://far-server:1234")
	assert.EqualError(t, connection.Open(), `failed to connect to NATS servers "[nats://far-server:1234]": nats: no servers available for connection`)
	connection.Close()
}

func TestConnectionWrap_Servers(t *testing.T) {
	connection, _ := newConnection("nats://far-server:1234")
	assert.Equal(t, []string{"nats://far-server:1234"}, connection.Servers())
}
