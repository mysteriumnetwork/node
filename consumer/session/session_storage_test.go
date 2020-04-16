/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	node_session "github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	errMock       = errors.New("error")
	mockSessionID = "sessionID"
	mockSession   = connection.Status{
		StartedAt:    time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
		SessionID:    node_session.ID(mockSessionID),
		ConsumerID:   identity.FromAddress("consumerID"),
		AccountantID: common.HexToAddress("0x00000000000000000000000000000000000000AC"),
		Proposal: market.ServiceProposal{
			ServiceDefinition: &StubServiceDefinition{},
			ServiceType:       "serviceType",
			ProviderID:        "providerID",
		},
	}
	mockStats   = connection.Statistics{BytesReceived: 100000, BytesSent: 50000}
	mockInvoice = crypto.Invoice{AgreementID: 10, AgreementTotal: 1000, TransactorFee: 10}
)

func TestSessionStorageGetAll(t *testing.T) {
	storer := &StubSessionStorer{}
	storage := NewSessionStorage(storer)
	sessions, err := storage.GetAll()
	assert.Nil(t, err)
	assert.True(t, storer.GetAllCalled)
	assert.Len(t, sessions, 0)
}

func TestSessionStorageGetAllReturnsError(t *testing.T) {
	storer := &StubSessionStorer{
		GetAllError: errMock,
	}
	storage := NewSessionStorage(storer)
	sessions, err := storage.GetAll()
	assert.NotNil(t, err)
	assert.True(t, storer.GetAllCalled)
	assert.Nil(t, sessions)
}

func TestSessionStorage_consumeEventEndedOK(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	storage.consumeSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: mockSession,
	})
	storage.consumeSessionStatisticsEvent(connection.AppEventConnectionStatistics{
		Stats:       mockStats,
		SessionInfo: mockSession,
	})
	storage.consumeSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionEndedStatus,
		SessionInfo: mockSession,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       node_session.ID("sessionID"),
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "Completed",
			Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
			DataStats:       mockStats,
			Invoice:         crypto.Invoice{},
		},
		storer.UpdatedObject,
	)
}

func TestSessionStorage_consumeEventConnectedOK(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.consumeSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: mockSession,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       node_session.ID("sessionID"),
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "New",
			Updated:         time.Time{},
			DataStats:       connection.Statistics{},
			Invoice:         crypto.Invoice{},
		},
		storer.SavedObject,
	)
}

func TestSessionStorage_consumeSessionSpendingEvent(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	storage.consumeSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: mockSession,
	})

	storage.consumeSessionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  "unknown",
		Invoice:    mockInvoice,
	})
	assert.Nil(t, storer.UpdatedObject)

	storage.consumeSessionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  mockSessionID,
		Invoice:    mockInvoice,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       node_session.ID("sessionID"),
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "New",
			Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
			DataStats:       connection.Statistics{},
			Invoice:         mockInvoice,
		},
		storer.UpdatedObject,
	)
}

// StubSessionStorer allows us to get all sessions, save and update them
type StubSessionStorer struct {
	SaveError     error
	SavedObject   interface{}
	UpdateError   error
	UpdatedObject interface{}
	GetAllCalled  bool
	GetAllError   error
}

func (sss *StubSessionStorer) Store(from string, object interface{}) error {
	sss.SavedObject = object
	return sss.SaveError
}

func (sss *StubSessionStorer) Update(from string, object interface{}) error {
	sss.UpdatedObject = object
	return sss.UpdateError
}

func (sss *StubSessionStorer) GetAllFrom(from string, array interface{}) error {
	sss.GetAllCalled = true
	return sss.GetAllError
}

type StubServiceDefinition struct{}

func (fs *StubServiceDefinition) GetLocation() market.Location {
	return market.Location{Country: "MU"}
}
