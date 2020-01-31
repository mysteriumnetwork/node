/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package firewall

import (
	"strings"
)

type iptablesExecResult struct {
	called bool
	output []string
	err    error
}

type iptablesExecMock struct {
	mocks map[string]iptablesExecResult
}

func (mce *iptablesExecMock) Exec(args ...string) ([]string, error) {
	key := argsToKey(args...)
	res := mce.mocks[key]
	res.called = true
	mce.mocks[key] = res
	return res.output, res.err
}

func (mce *iptablesExecMock) VerifyCalledWithArgs(args ...string) bool {
	key := argsToKey(args...)
	return mce.mocks[key].called
}

func argsToKey(args ...string) string {
	return strings.Join(args, " ")
}
