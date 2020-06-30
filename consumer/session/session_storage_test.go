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
	session_node "github.com/mysteriumnetwork/node/session"
	session_event "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	serviceSessionMock = session_event.SessionContext{
		ID:           "session1",
		StartedAt:    time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		ConsumerID:   identity.FromAddress("consumer1"),
		AccountantID: common.HexToAddress("0x00000000000000000000000000000000000000AC"),
		Proposal: market.ServiceProposal{
			ServiceDefinition: &StubServiceDefinition{},
			ServiceType:       "serviceType",
			ProviderID:        "providerID",
		},
	}
	connectionSessionMock = connection.Status{
		StartedAt:    time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
		SessionID:    session_node.ID("sessionID"),
		ConsumerID:   identity.FromAddress("consumerID"),
		AccountantID: common.HexToAddress("0x00000000000000000000000000000000000000AC"),
		Proposal: market.ServiceProposal{
			ServiceDefinition: &StubServiceDefinition{},
			ServiceType:       "serviceType",
			ProviderID:        "providerID",
		},
	}
	connectionStatsMock   = connection.Statistics{BytesReceived: 100000, BytesSent: 50000}
	connectionInvoiceMock = crypto.Invoice{AgreementID: 10, AgreementTotal: 1000, TransactorFee: 10}
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
		GetAllError: errors.New("error"),
	}
	storage := NewSessionStorage(storer)
	sessions, err := storage.GetAll()
	assert.NotNil(t, err)
	assert.True(t, storer.GetAllCalled)
	assert.Nil(t, sessions)
}

func TestSessionStorage_consumeServiceSessionsEvent(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	storage.consumeServiceSessionEvent(session_event.AppEventSession{
		Status:  session_event.CreatedStatus,
		Session: serviceSessionMock,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       session_node.ID("session1"),
			Direction:       "Provider",
			ConsumerID:      identity.FromAddress("consumer1"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
			Status:          "New",
		},
		storer.SavedObject,
	)

	storage.consumeServiceSessionStatisticsEvent(session_event.AppEventDataTransferred{
		ID:   serviceSessionMock.ID,
		Up:   123,
		Down: 1234,
	})
	storage.consumeServiceSessionEarningsEvent(session_event.AppEventTokensEarned{
		SessionID: serviceSessionMock.ID,
		Total:     12,
	})
	storage.consumeServiceSessionEvent(session_event.AppEventSession{
		Status:  session_event.RemovedStatus,
		Session: serviceSessionMock,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       session_node.ID("session1"),
			Direction:       "Provider",
			ConsumerID:      identity.FromAddress("consumer1"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
			Status:          "Completed",
			Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
			DataSent:        1234,
			DataReceived:    123,
			Tokens:          12,
		},
		storer.UpdatedObject,
	)
}

func TestSessionStorage_consumeEventEndedOK(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})
	storage.consumeConnectionStatisticsEvent(connection.AppEventConnectionStatistics{
		Stats:       connectionStatsMock,
		SessionInfo: connectionSessionMock,
	})
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionEndedStatus,
		SessionInfo: connectionSessionMock,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       session_node.ID("sessionID"),
			Direction:       "Consumer",
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "Completed",
			Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
			DataSent:        connectionStatsMock.BytesSent,
			DataReceived:    connectionStatsMock.BytesReceived,
		},
		storer.UpdatedObject,
	)
}

func TestSessionStorage_consumeEventConnectedOK(t *testing.T) {
	storer := &StubSessionStorer{}

	storage := NewSessionStorage(storer)
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       session_node.ID("sessionID"),
			Direction:       "Consumer",
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "New",
			Updated:         time.Time{},
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
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})

	storage.consumeConnectionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  "unknown",
		Invoice:    connectionInvoiceMock,
	})
	assert.Nil(t, storer.UpdatedObject)

	storage.consumeConnectionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  "sessionID",
		Invoice:    connectionInvoiceMock,
	})
	assert.Equal(
		t,
		&History{
			SessionID:       session_node.ID("sessionID"),
			Direction:       "Consumer",
			ConsumerID:      identity.FromAddress("consumerID"),
			AccountantID:    "0x00000000000000000000000000000000000000AC",
			ProviderID:      identity.FromAddress("providerID"),
			ServiceType:     "serviceType",
			ProviderCountry: "MU",
			Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
			Status:          "New",
			Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
			Tokens:          connectionInvoiceMock.AgreementTotal,
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
