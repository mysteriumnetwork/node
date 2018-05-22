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

package management

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAddress_String(t *testing.T) {
	addr := Address{"127.0.0.1", 8080}
	assert.Equal(t, "127.0.0.1:8080", addr.String())
}

func TestGetAddressFromString(t *testing.T) {
	addr, err := GetPortAndAddressFromString("127.0.0.1:8080")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", addr.IP)
	assert.Equal(t, 8080, addr.Port)
}

func TestGetAddressFromStringFails(t *testing.T) {
	addr, err := GetPortAndAddressFromString("127.0.0.1::")
	assert.Equal(t, "Failed to parse port number.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("::")
	assert.Equal(t, "Failed to parse port number.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("127.0.0.1")
	assert.Equal(t, "Failed to parse address string.", err.Error())
	assert.Nil(t, addr)

	addr, err = GetPortAndAddressFromString("")
	assert.Equal(t, "Failed to parse address string.", err.Error())
	assert.Nil(t, addr)
}
