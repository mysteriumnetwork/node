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

package migrations

import (
	"sort"
	"testing"
	"time"

	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"

	"github.com/stretchr/testify/assert"
)

var (
	sessionID       = node_session.ID("sessionID")
	providerID      = identity.FromAddress("providerID")
	serviceType     = "serviceType"
	timeStarted     = time.Now().UTC()
	timeUpdated     = time.Now().UTC()
	providerCountry = "providerCountry"
	bucketName      = "session-history"
	statsMock       = connection.Statistics{
		BytesSent:     1,
		BytesReceived: 1,
	}
	oldSessionMock = Session{
		SessionID:       sessionID,
		ProviderID:      providerID,
		ServiceType:     serviceType,
		ProviderCountry: providerCountry,
		Started:         timeStarted,
		Updated:         timeUpdated,
		DataStats:       statsMock,
	}
)

func TestSessionToHistoryMigrationWithNoData(t *testing.T) {
	file, db := boltdbtest.CreateDB(t)
	defer boltdbtest.CleanupDB(t, file, db)

	err := MigrateSessionToHistory(db)
	assert.Nil(t, err)

	histories := []consumer_session.History{}
	err = db.From(bucketName).All(&histories)

	assert.Nil(t, err)
	assert.Len(t, histories, 0)
}

func TestSessionToHistoryMigrationWithData(t *testing.T) {
	file, db := boltdbtest.CreateDB(t)
	defer boltdbtest.CleanupDB(t, file, db)

	tx, err := db.Begin(true)
	assert.Nil(t, err)

	mockNewSession := oldSessionMock
	mockNewSession.Status = 0
	err = tx.Save(&mockNewSession)
	assert.Nil(t, err)

	mockCompletedSession := oldSessionMock
	mockCompletedSession.Status = 1
	mockCompletedSession.SessionID = node_session.ID("sessionID2")
	mockCompletedSession.Started = time.Now().UTC()
	mockCompletedSession.Updated = time.Now().UTC()
	err = tx.Save(&mockCompletedSession)
	assert.Nil(t, err)

	err = tx.Commit()
	assert.Nil(t, err)

	err = MigrateSessionToHistory(db)
	assert.Nil(t, err)

	histories := []consumer_session.History{}
	err = db.From(bucketName).All(&histories)

	assert.Nil(t, err)
	assert.Len(t, histories, 2)

	sort.Slice(histories, func(i, j int) bool {
		return histories[i].Started.Before(histories[j].Started)
	})

	assert.Equal(t, histories[0].Status, consumer_session.SessionStatusNew)
	assert.Equal(t, histories[1].Status, consumer_session.SessionStatusCompleted)
}
