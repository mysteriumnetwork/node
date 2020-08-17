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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/rs/zerolog/log"
)

// Session structure holds all required information about current session between service consumer and provider.
type Session struct {
	ID               session.ID
	ConsumerID       identity.Identity
	ConsumerLocation market.Location
	AccountantID     common.Address
	Proposal         market.ServiceProposal
	ServiceID        string
	CreatedAt        time.Time
	request          *pb.SessionRequest
	done             chan struct{}
	cleanupLock      sync.Mutex
	cleanup          []func() error
	tracer           *trace.Tracer
}

// Close ends session.
func (s *Session) Close() {
	close(s.done)

	s.cleanupLock.Lock()
	defer s.cleanupLock.Unlock()

	for i := len(s.cleanup) - 1; i >= 0; i-- {
		log.Trace().Msgf("Session cleaning up: (%v/%v)", i+1, len(s.cleanup))
		err := s.cleanup[i]()
		if err != nil {
			log.Warn().Err(err).Msg("Cleanup error")
		}
	}
	s.cleanup = nil
}

// Done returns readonly done channel.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) addCleanup(fn func() error) {
	s.cleanupLock.Lock()
	defer s.cleanupLock.Unlock()

	s.cleanup = append(s.cleanup, fn)
}

func (s *Session) toEvent(status event.Status) event.AppEventSession {
	return event.AppEventSession{
		Status: status,
		Service: event.ServiceContext{
			ID: s.ServiceID,
		},
		Session: event.SessionContext{
			ID:               string(s.ID),
			StartedAt:        s.CreatedAt,
			ConsumerID:       s.ConsumerID,
			ConsumerLocation: s.ConsumerLocation,
			AccountantID:     s.AccountantID,
			Proposal:         s.Proposal,
		},
	}
}

// NewSession creates a blank new session with an ID.
func NewSession(service *Instance, request *pb.SessionRequest) (*Session, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	var consumerLocation market.Location
	if location := request.GetConsumer().GetLocation(); location != nil {
		consumerLocation.Country = location.GetCountry()
	}

	return &Session{
		ID:               session.ID(uid.String()),
		ConsumerID:       identity.FromAddress(request.GetConsumer().GetId()),
		ConsumerLocation: consumerLocation,
		AccountantID:     common.HexToAddress(request.GetConsumer().GetAccountantID()),
		Proposal:         service.Proposal,
		ServiceID:        string(service.ID),
		CreatedAt:        time.Now().UTC(),
		request:          request,
		done:             make(chan struct{}),
		cleanup:          make([]func() error, 0),
		tracer:           trace.NewTracer(),
	}, nil
}
