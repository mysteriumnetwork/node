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

package nat

// NewService returns linux os specific nat service based on ip tables
func NewService() NATService {
	return &serviceIPTables{
		ipForward: serviceIPForward{
			CommandEnable:  []string{"sudo", "/sbin/sysctl", "-w", "net.ipv4.ip_forward=1"},
			CommandDisable: []string{"sudo", "/sbin/sysctl", "-w", "net.ipv4.ip_forward=0"},
			CommandRead:    []string{"/sbin/sysctl", "-n", "net.ipv4.ip_forward"},
		},
		rules: make(map[RuleForwarding]struct{}),
	}
}
