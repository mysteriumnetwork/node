//+build !windows

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

package firewall

// AddInboundRule adds new inbound rule to the platform specific firewall.
func AddInboundRule(proto string, port int) error {
	// TODO adding firewall rules should be implemented for every platform.
	return nil
}

// RemoveInboundRule removes inbound rule from the platform specific firewall.
func RemoveInboundRule(proto string, port int) error {
	// TODO adding firewall rules should be implemented for every platform.
	return nil
}
