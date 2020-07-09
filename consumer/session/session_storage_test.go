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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
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
	// given
	sessionExpected := History{
		SessionID:       session_node.ID("session1"),
		Direction:       "Provided",
		ConsumerID:      identity.FromAddress("consumer1"),
		AccountantID:    "0x00000000000000000000000000000000000000AC",
		ProviderID:      identity.FromAddress("providerID"),
		ServiceType:     "serviceType",
		ProviderCountry: "MU",
		Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		Status:          "New",
	}
	storage, storageCleanup := newStorageWithSessions(sessionExpected)
	defer storageCleanup()

	// when
	sessions, err := storage.GetAll()

	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{sessionExpected}, sessions)
}

func TestSessionStorage_consumeServiceSessionsEvent(t *testing.T) {
	// given
	storage, storageCleanup := newStorage()
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	defer storageCleanup()

	// when
	storage.consumeServiceSessionEvent(session_event.AppEventSession{
		Status:  session_event.CreatedStatus,
		Session: serviceSessionMock,
	})
	// then
	sessions, err := storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("session1"),
				Direction:       "Provided",
				ConsumerID:      identity.FromAddress("consumer1"),
				AccountantID:    "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
				Status:          "New",
			},
		},
		sessions,
	)

	// when
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
	// then
	sessions, err = storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("session1"),
				Direction:       "Provided",
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
		},
		sessions,
	)
}

func TestSessionStorage_consumeEventEndedOK(t *testing.T) {
	// given
	storage, storageCleanup := newStorage()
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	defer storageCleanup()

	// when
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

	// then
	sessions, err := storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("sessionID"),
				Direction:       "Consumed",
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
		},
		sessions,
	)
}

func TestSessionStorage_consumeEventConnectedOK(t *testing.T) {
	// given
	storage, storageCleanup := newStorage()
	defer storageCleanup()

	// when
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})

	// then
	sessions, err := storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("sessionID"),
				Direction:       "Consumed",
				ConsumerID:      identity.FromAddress("consumerID"),
				AccountantID:    "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
				Status:          "New",
				Updated:         time.Time{},
			},
		},
		sessions,
	)
}

func TestSessionStorage_consumeSessionSpendingEvent(t *testing.T) {
	// given
	storage, storageCleanup := newStorage()
	storage.timeGetter = func() time.Time {
		return time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	}
	defer storageCleanup()

	// when
	storage.consumeConnectionSessionEvent(connection.AppEventConnectionSession{
		Status:      connection.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})
	storage.consumeConnectionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  "unknown",
		Invoice:    connectionInvoiceMock,
	})
	// then
	sessions, err := storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("sessionID"),
				Direction:       "Consumed",
				ConsumerID:      identity.FromAddress("consumerID"),
				AccountantID:    "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
				Status:          "New",
				Updated:         time.Time{},
			},
		},
		sessions,
	)

	// when
	storage.consumeConnectionSpendingEvent(event.AppEventInvoicePaid{
		ConsumerID: identity.FromAddress("me"),
		SessionID:  "sessionID",
		Invoice:    connectionInvoiceMock,
	})
	// then
	sessions, err = storage.GetAll()
	assert.Nil(t, err)
	assert.Equal(
		t,
		[]History{
			{
				SessionID:       session_node.ID("sessionID"),
				Direction:       "Consumed",
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
		},
		sessions,
	)
}

func newStorage() (*Storage, func()) {
	dir, err := ioutil.TempDir("", "sessionStorageTest")
	if err != nil {
		panic(err)
	}

	db, err := boltdb.NewStorage(dir)
	if err != nil {
		panic(err)
	}

	return NewSessionStorage(db), func() {
		err := db.Close()
		if err != nil {
			panic(err)
		}

		err = os.RemoveAll(dir)
		if err != nil {
			panic(err)
		}
	}
}

func newStorageWithSessions(sessions ...History) (*Storage, func()) {
	storage, storageCleanup := newStorage()
	for _, session := range sessions {
		err := storage.storage.Store(sessionStorageBucketName, &session)
		if err != nil {
			panic(err)
		}
	}
	return storage, storageCleanup
}

type StubServiceDefinition struct{}

func (fs *StubServiceDefinition) GetLocation() market.Location {
	return market.Location{Country: "MU"}
}
