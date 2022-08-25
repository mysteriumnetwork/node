/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

type serviceNoop struct{}

// Setup sets NAT/Firewall rules for the given NATOptions.
func (svc *serviceNoop) Setup(opts Options) (appliedRules []interface{}, err error) {
	return nil, nil
}

// Del removes given NAT/Firewall rules that were previously set up.
func (svc *serviceNoop) Del(rules []interface{}) error {
	return nil
}

// Enable enables NAT service.
func (svc *serviceNoop) Enable() error {
	return nil
}

// Disable disables NAT service and deletes all rules.
func (svc *serviceNoop) Disable() error {
	return nil
}
