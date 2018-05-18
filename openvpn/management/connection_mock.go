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
	"fmt"
)

// MockConnection is mock openvpn management interface used for middleware testing
type MockConnection struct {
	WrittenLines      []string
	LastLine          string
	CommandResult     string
	MultilineResponse []string
}

func (conn *MockConnection) SingleLineCommand(format string, args ...interface{}) (string, error) {
	conn.LastLine = fmt.Sprintf(format, args...)
	conn.WrittenLines = append(conn.WrittenLines, conn.LastLine)
	return conn.CommandResult, nil
}

func (conn *MockConnection) MultiLineCommand(format string, args ...interface{}) (string, []string, error) {
	_, _ = conn.SingleLineCommand(format, args...)
	return conn.CommandResult, conn.MultilineResponse, nil
}
