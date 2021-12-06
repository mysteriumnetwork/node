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

package registry

import (
	"time"

	"github.com/mysteriumnetwork/node/identity"
)

// RegistrationStatus represents all the possible registration statuses
type RegistrationStatus int

const (
	// Registered represents a registration
	Registered RegistrationStatus = iota
	// Unregistered represents an unregistered identity
	Unregistered
	// InProgress shows that registration is still in progress
	InProgress
	// RegistrationError shows that an error occurred during registration
	RegistrationError
	// Unknown is returned when there was an error fetching registration status and it is now
	// impossible to determine. A request should be retried to get the status again.
	Unknown
)

// String converts registration to human readable notation
func (rs RegistrationStatus) String() string {
	return [...]string{
		"Registered",
		"Unregistered",
		"InProgress",
		"RegistrationError",
		"Unknown",
	}[rs]
}

// Registered returns flag if registration is in successful status
func (rs RegistrationStatus) Registered() bool {
	switch rs {
	case Registered:
		return true
	default:
		return false
	}
}

// StoredRegistrationStatus represents a registration status that is being stored in our local DB
type StoredRegistrationStatus struct {
	RegistrationStatus  RegistrationStatus
	Identity            identity.Identity
	ChainID             int64
	RegistrationRequest IdentityRegistrationRequest
	UpdatedAt           time.Time
}

// FromEvent constructs a stored registration status from transactor.IdentityRegistrationRequest
func (srs StoredRegistrationStatus) FromEvent(status RegistrationStatus, ev IdentityRegistrationRequest) StoredRegistrationStatus {
	return StoredRegistrationStatus{
		RegistrationStatus:  status,
		RegistrationRequest: ev,
		Identity:            identity.FromAddress(ev.Identity),
		ChainID:             ev.ChainID,
	}
}

// AppTopicIdentityRegistration represents the registration event topic.
const AppTopicIdentityRegistration = "registration_event_topic"

// AppEventIdentityRegistration represents the registration event payload.
type AppEventIdentityRegistration struct {
	ID      identity.Identity
	Status  RegistrationStatus
	ChainID int64
}
