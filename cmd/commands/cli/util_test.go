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

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseCommandAndArgs(t *testing.T) {
	for _, test := range []struct {
		args string

		expectedCmd  string
		expectedArgs []string
	}{
		{
			args: "cmd",

			expectedCmd:  "cmd",
			expectedArgs: []string{},
		},
		{
			args: "cmd arg1 arg2",

			expectedCmd:  "cmd",
			expectedArgs: []string{"arg1", "arg2"},
		},
		{
			args: `cmd "arg1 arg2" arg3`,

			expectedCmd:  "cmd",
			expectedArgs: []string{"arg1 arg2", "arg3"},
		},
		{
			args: `cmd 'arg1 arg2' arg3`,

			expectedCmd:  "cmd",
			expectedArgs: []string{"arg1 arg2", "arg3"},
		},
		{
			args: `cmd 'arg1' "arg2" arg3`,

			expectedCmd:  "cmd",
			expectedArgs: []string{"arg1", "arg2", "arg3"},
		},
		{
			args: `cmd 'arg1 arg2' "arg3 arg4"`,

			expectedCmd:  "cmd",
			expectedArgs: []string{"arg1 arg2", "arg3 arg4"},
		},
	} {
		gotCmd, gotArgs, err := parseCommandAndArgs(test.args)
		assert.NoError(t, err)

		assert.Equal(t, test.expectedCmd, gotCmd)
		assert.Equal(t, test.expectedArgs, gotArgs)
	}
}
