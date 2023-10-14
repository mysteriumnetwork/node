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
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
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
		ID:         "session1",
		StartedAt:  time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		ConsumerID: identity.FromAddress("consumer1"),
		HermesID:   common.HexToAddress("0x00000000000000000000000000000000000000AC"),
		Proposal: market.ServiceProposal{
			Location:    stubLocation,
			ServiceType: "serviceType",
			ProviderID:  "providerID",
		},
	}
	connectionSessionMock = connectionstate.Status{
		StartedAt:  time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
		SessionID:  session_node.ID("sessionID"),
		ConsumerID: identity.FromAddress("consumerID"),
		HermesID:   common.HexToAddress("0x00000000000000000000000000000000000000AC"),
		Proposal: proposal.PricedServiceProposal{
			ServiceProposal: market.ServiceProposal{
				Location:    stubLocation,
				ServiceType: "serviceType",
				ProviderID:  "providerID",
			},
		},
	}
	connectionStatsMock   = connectionstate.Statistics{BytesReceived: 100000, BytesSent: 50000}
	connectionInvoiceMock = crypto.Invoice{AgreementID: big.NewInt(10), AgreementTotal: big.NewInt(1000), TransactorFee: big.NewInt(10)}
)

func TestSessionStorage_GetAll(t *testing.T) {
	// given
	sessionExpected := History{
		SessionID:       session_node.ID("session1"),
		Direction:       "Provided",
		ConsumerID:      identity.FromAddress("consumer1"),
		HermesID:        "0x00000000000000000000000000000000000000AC",
		ProviderID:      identity.FromAddress("providerID"),
		ServiceType:     "serviceType",
		ProviderCountry: "MU",
		Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		Status:          "New",
	}
	storage, storageCleanup := newStorageWithSessions(sessionExpected)
	defer storageCleanup()

	// when
	result, err := storage.GetAll()
	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{sessionExpected}, result)
}

func TestSessionStorage_List(t *testing.T) {
	// given
	session1Expected := History{
		SessionID: session_node.ID("session1"),
		Started:   time.Date(2020, 6, 17, 0, 0, 1, 0, time.UTC),
	}
	session2Expected := History{
		SessionID: session_node.ID("session2"),
		Started:   time.Date(2020, 6, 17, 0, 0, 2, 0, time.UTC),
	}
	storage, storageCleanup := newStorageWithSessions(session1Expected, session2Expected)
	defer storageCleanup()

	// when
	result, err := storage.List(NewFilter())
	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{session2Expected, session1Expected}, result)
}

func TestSessionStorage_ListFiltersDirection(t *testing.T) {
	// given
	sessionExpected := History{
		SessionID: session_node.ID("session1"),
		Direction: DirectionProvided,
	}
	storage, storageCleanup := newStorageWithSessions(sessionExpected)
	defer storageCleanup()

	// when
	result, err := storage.List(NewFilter())
	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{sessionExpected}, result)

	// when
	result, err = storage.List(NewFilter().SetDirection(DirectionProvided))
	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{sessionExpected}, result)

	// when
	result, err = storage.List(NewFilter().SetDirection(DirectionConsumed))
	// then
	assert.Nil(t, err)
	assert.Equal(t, []History{}, result)
}

func TestSessionStorage_Stats(t *testing.T) {
	// given
	sessionExpected := History{
		SessionID:    session_node.ID("session1"),
		Direction:    "Provided",
		ConsumerID:   identity.FromAddress("consumer1"),
		DataSent:     1234,
		DataReceived: 123,
		Tokens:       big.NewInt(12),
		Started:      time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		Updated:      time.Date(2020, 6, 17, 10, 11, 32, 0, time.UTC),
		Status:       "New",
	}
	storage, storageCleanup := newStorageWithSessions(sessionExpected)
	defer storageCleanup()

	// when
	result, err := storage.Stats(NewFilter())
	// then
	assert.Nil(t, err)
	assert.Equal(
		t,
		Stats{
			Count: 1,
			ConsumerCounts: map[identity.Identity]int{
				identity.FromAddress("consumer1"): 1,
			},
			SumDataSent:     1234,
			SumDataReceived: 123,
			SumTokens:       big.NewInt(12),
			SumDuration:     20 * time.Second,
		},
		result,
	)

	// when
	result, err = storage.Stats(NewFilter().SetDirection(DirectionConsumed))
	// then
	assert.Nil(t, err)
	assert.Equal(t, NewStats(), result)
}

func TestSessionStorage_StatsByDay(t *testing.T) {
	// given
	sessionExpected := History{
		SessionID:    session_node.ID("session1"),
		Direction:    "Provided",
		ConsumerID:   identity.FromAddress("consumer1"),
		DataSent:     1234,
		DataReceived: 123,
		Tokens:       big.NewInt(12),
		Started:      time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
		Updated:      time.Date(2020, 6, 17, 10, 11, 32, 0, time.UTC),
		Status:       "New",
	}
	storage, storageCleanup := newStorageWithSessions(sessionExpected)
	defer storageCleanup()

	// when
	filter := NewFilter().
		SetStartedFrom(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)).
		SetStartedTo(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC))
	result, err := storage.StatsByDay(filter)
	// then
	assert.Nil(t, err)
	assert.Equal(
		t,
		map[time.Time]Stats{
			time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC): NewStats(),
		},
		result,
	)

	// when
	filter = NewFilter().
		SetStartedFrom(time.Date(2020, 6, 17, 0, 0, 0, 0, time.UTC)).
		SetStartedTo(time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC))
	result, err = storage.StatsByDay(filter)
	// then
	assert.Nil(t, err)
	assert.Equal(
		t,
		map[time.Time]Stats{
			time.Date(2020, 6, 17, 0, 0, 0, 0, time.UTC): {
				Count: 1,
				ConsumerCounts: map[identity.Identity]int{
					identity.FromAddress("consumer1"): 1,
				},
				SumDataSent:     1234,
				SumDataReceived: 123,
				SumTokens:       big.NewInt(12),
				SumDuration:     20 * time.Second,
			},
			time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC): NewStats(),
		},
		result,
	)

	// when
	filter = NewFilter().
		SetStartedFrom(time.Date(2020, 6, 17, 0, 0, 0, 0, time.UTC)).
		SetStartedTo(time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC)).
		SetDirection(DirectionConsumed)
	result, err = storage.StatsByDay(filter)
	// then
	assert.Nil(t, err)
	assert.Equal(
		t,
		map[time.Time]Stats{
			time.Date(2020, 6, 17, 0, 0, 0, 0, time.UTC): NewStats(),
			time.Date(2020, 6, 18, 0, 0, 0, 0, time.UTC): NewStats(),
		},
		result,
	)
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
				Status:          "New",
				Tokens:          new(big.Int),
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
		Total:     big.NewInt(12),
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 6, 17, 10, 11, 12, 0, time.UTC),
				Status:          "Completed",
				Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
				DataSent:        1234,
				DataReceived:    123,
				Tokens:          big.NewInt(12),
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
	storage.consumeConnectionSessionEvent(connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionCreatedStatus,
		SessionInfo: connectionSessionMock,
	})
	storage.consumeConnectionStatisticsEvent(connectionstate.AppEventConnectionStatistics{
		Stats:       connectionStatsMock,
		SessionInfo: connectionSessionMock,
	})
	storage.consumeConnectionSessionEvent(connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionEndedStatus,
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
				Status:          "Completed",
				Updated:         time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC),
				DataSent:        connectionStatsMock.BytesSent,
				DataReceived:    connectionStatsMock.BytesReceived,
				Tokens:          big.NewInt(0),
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
	storage.consumeConnectionSessionEvent(connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionCreatedStatus,
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
				Status:          "New",
				Updated:         time.Time{},
				Tokens:          big.NewInt(0),
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
	storage.consumeConnectionSessionEvent(connectionstate.AppEventConnectionSession{
		Status:      connectionstate.SessionCreatedStatus,
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
				ProviderID:      identity.FromAddress("providerID"),
				ServiceType:     "serviceType",
				ProviderCountry: "MU",
				Started:         time.Date(2020, 4, 1, 10, 11, 12, 0, time.UTC),
				Status:          "New",
				Updated:         time.Time{},
				Tokens:          big.NewInt(0),
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
				HermesID:        "0x00000000000000000000000000000000000000AC",
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
	dir, err := os.MkdirTemp("", "sessionStorageTest")
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

var stubLocation = market.Location{Country: "MU"}
