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

package management

// Management packages contains all functionality related to openvpn management interface
// See https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html

// Connection represents openvpn management interface abstraction for middlewares to be able to send commands to openvpn process
type Connection interface {
	SingleLineCommand(template string, args ...interface{}) (string, error)
	MultiLineCommand(template string, args ...interface{}) (string, []string, error)
}

// Middleware used to control openvpn process through management interface
// It's guaranteed that ConsumeLine callback will be called AFTER Start callback is finished
// Connection passed on Stop callback can be already closed - expect errors when sending commands
// For efficiency and simplicity purposes ConsumeLine for each middleware is called from the same goroutine which
// consumes events from channel - avoid long running operations at all costs
type Middleware interface {
	Start(Connection) error
	Stop(Connection) error
	ConsumeLine(line string) (consumed bool, err error)
}
