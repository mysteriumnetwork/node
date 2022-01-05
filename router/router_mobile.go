//go:build android

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

package router

import (
	"net"
)

type manager struct{}

// NewManager creates a new instance of service that maintain routing table to match current state.
func NewManager() *manager {
	return &manager{}
}

func (m *manager) ExcludeIP(ip net.IP) error {
	return nil
}

func (m *manager) RemoveExcludedIP(ip net.IP) error {
	return nil
}

func (m *manager) Stop() {}

func (m *manager) Clean() error {
	return nil
}
