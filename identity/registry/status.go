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
	// RegisteredConsumer represents a registration with 0 stake
	RegisteredConsumer RegistrationStatus = iota
	// RegisteredProvider represents a registration with stake > 0
	RegisteredProvider
	// Unregistered represents an unregistered identity
	Unregistered
	// InProgress shows that registration is still in progress
	InProgress
	// Promoting shows that a consumer is being promoted to provider
	Promoting
	// RegistrationError shows that an error occurred during registration
	RegistrationError
)

func (rs RegistrationStatus) String() string {
	return [...]string{
		"RegisteredConsumer",
		"RegisteredProvider",
		"Unregistered",
		"InProgress",
		"Promoting",
		"RegistrationError",
	}[rs]
}

// StoredRegistrationStatus represents a registration status that is being stored in our local DB
type StoredRegistrationStatus struct {
	RegistrationStatus  RegistrationStatus
	Identity            identity.Identity `storm:"id"`
	RegistrationRequest IdentityRegistrationRequest
	UpdatedAt           time.Time
}

// FromEvent constructs a stored registration status from transactor.IdentityRegistrationRequest
func (srs StoredRegistrationStatus) FromEvent(status RegistrationStatus, ev IdentityRegistrationRequest) StoredRegistrationStatus {
	return StoredRegistrationStatus{
		RegistrationStatus:  status,
		RegistrationRequest: ev,
		Identity:            identity.FromAddress(ev.Identity),
	}
}

// AppTopicRegistration represents the registration event topic
const AppTopicRegistration = "registration_event_topic"

// RegistrationEventPayload represents the registration event payload
type RegistrationEventPayload struct {
	ID     identity.Identity
	Status RegistrationStatus
}
