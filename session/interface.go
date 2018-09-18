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

import "github.com/mysteriumnetwork/node/identity"

// SessionID represents session id type
type SessionID string

// Session structure holds all required information about current session between service consumer and provider
type Session struct {
	ID         SessionID
	Config     ServiceConfiguration
	ConsumerID identity.Identity
}

// Generator defines method for session id generation
type Generator interface {
	Generate() SessionID
}

// Manager defines methods for session management
type Manager interface {
	Create(identity.Identity) (Session, error)
}
