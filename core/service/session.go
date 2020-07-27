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

package service

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/event"
)

// Session structure holds all required information about current session between service consumer and provider.
type Session struct {
	ID           session.ID
	ConsumerID   identity.Identity
	AccountantID common.Address
	Proposal     market.ServiceProposal
	ServiceID    string
	CreatedAt    time.Time
	done         chan struct{}
}

// Done returns readonly done channel.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s Session) toEvent(status event.Status) event.AppEventSession {
	return event.AppEventSession{
		Status: status,
		Service: event.ServiceContext{
			ID: s.ServiceID,
		},
		Session: event.SessionContext{
			ID:           string(s.ID),
			StartedAt:    s.CreatedAt,
			ConsumerID:   s.ConsumerID,
			AccountantID: s.AccountantID,
			Proposal:     s.Proposal,
		},
	}
}

// NewSession creates a blank new session with an ID.
func NewSession() (*Session, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	return &Session{ID: session.ID(uid.String())}, nil
}
