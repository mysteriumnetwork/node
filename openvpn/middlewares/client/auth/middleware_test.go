/*
 * Copyright (C) 2018 The Mysterium Network Authors
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

package auth

import (
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
)

func auth() (string, string, error) {
	return "testuser", "testpassword", nil
}

func Test_Factory(t *testing.T) {
	middleware := NewMiddleware(auth)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">SOME_LINE_DELIVERED"},
		{">ANOTHER_LINE_DELIVERED"},
		{">PASSWORD"},
	}
	middleware := NewMiddleware(auth)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
	}
}

func Test_ConsumeLineTakes(t *testing.T) {
	passwordRequest := ">PASSWORD:Need 'Auth' username/password"

	middleware := NewMiddleware(auth)
	mockCmdWriter := &management.MockConnection{}
	middleware.Start(mockCmdWriter)

	consumed, err := middleware.ConsumeLine(passwordRequest)
	assert.NoError(t, err)
	assert.True(t, consumed)
	assert.Equal(t,
		mockCmdWriter.WrittenLines,
		[]string{
			"password 'Auth' testpassword",
			"username 'Auth' testuser",
		},
	)
}
