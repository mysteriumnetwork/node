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

package session

import "github.com/mysterium/node/identity"

// ManagerFake represents fake manager usually useful in tests
type ManagerFake struct{}

var fakeConfig = struct {
	Param1 string
	Param2 int
}{
	"string-param",
	123,
}

// Create function creates and returns fake session
func (manager *ManagerFake) Create(peerID identity.Identity) (Session, error) {
	return Session{"new-id", fakeConfig, peerID}, nil
}

// FindSession always returns empty session and signals that session is not found
func (manager *ManagerFake) FindSession(id SessionID) (Session, bool) {
	return Session{}, false
}
