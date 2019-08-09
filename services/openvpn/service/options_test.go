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

package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseJSONOptions_HandlesNil(t *testing.T) {
	options, err := ParseJSONOptions(nil)

	assert.NoError(t, err)
	assert.Equal(t, defaultOptions, options)
}

func Test_ParseJSONOptions_HandlesEmptyRequest(t *testing.T) {
	request := json.RawMessage(`{}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, defaultOptions, options)
}

func Test_ParseJSONOptions_ValidRequest(t *testing.T) {
	request := json.RawMessage(`{"port": 1123, "protocol": "udp", "subnet": "10.10.10.0", "netmask": "255.255.255.0"}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, Options{Protocol: "udp", Port: 1123, Subnet: "10.10.10.0", Netmask: "255.255.255.0"}, options)
}
