/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

import (
	"time"

	"github.com/asdine/storm/v3/q"
	"github.com/mysteriumnetwork/node/identity"
)

// NewFilter creates instance of filter.
func NewFilter() *Filter {
	return &Filter{}
}

// Filter defines all flags for session filtering in session storage.
type Filter struct {
	StartedFrom *time.Time
	StartedTo   *time.Time
	Direction   *string
	ConsumerID  *identity.Identity
	HermesID    *string
	ProviderID  *identity.Identity
	ServiceType *string
	Status      *string
}

// SetStartedFrom filters fetched sessions from given time.
func (f *Filter) SetStartedFrom(from time.Time) *Filter {
	from = from.UTC()
	f.StartedFrom = &from
	return f
}

// SetStartedTo filters fetched sessions to given time.
func (f *Filter) SetStartedTo(to time.Time) *Filter {
	to = to.UTC()
	f.StartedTo = &to
	return f
}

// SetDirection filters fetched sessions by direction.
func (f *Filter) SetDirection(direction string) *Filter {
	f.Direction = &direction
	return f
}

// SetConsumerID filters fetched sessions by consumer.
func (f *Filter) SetConsumerID(id identity.Identity) *Filter {
	f.ConsumerID = &id
	return f
}

// SetHermesID filters fetched sessions by hermes.
func (f *Filter) SetHermesID(hermesID string) *Filter {
	f.HermesID = &hermesID
	return f
}

// SetProviderID filters fetched sessions by provider.
func (f *Filter) SetProviderID(id identity.Identity) *Filter {
	f.ProviderID = &id
	return f
}

// SetServiceType filters fetched sessions by service type.
func (f *Filter) SetServiceType(serviceType string) *Filter {
	f.ServiceType = &serviceType
	return f
}

// SetStatus filters fetched sessions by status.
func (f *Filter) SetStatus(status string) *Filter {
	f.Status = &status
	return f
}

func (f *Filter) toMatcher() q.Matcher {
	where := make([]q.Matcher, 0)
	if f.StartedFrom != nil {
		where = append(where, q.Gte("Started", *f.StartedFrom))
	}
	if f.StartedTo != nil {
		where = append(where, q.Lte("Started", *f.StartedTo))
	}
	if f.Direction != nil {
		where = append(where, q.Eq("Direction", *f.Direction))
	}
	if f.ConsumerID != nil {
		where = append(where, q.Eq("ConsumerID", *f.ConsumerID))
	}
	if f.HermesID != nil {
		where = append(where, q.Eq("HermesID", *f.HermesID))
	}
	if f.ProviderID != nil {
		where = append(where, q.Eq("ProviderID", *f.ProviderID))
	}
	if f.ServiceType != nil {
		where = append(where, q.Eq("ServiceType", *f.ServiceType))
	}
	if f.Status != nil {
		where = append(where, q.Eq("Status", *f.Status))
	}
	return q.And(where...)
}
