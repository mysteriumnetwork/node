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

package endpoints

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer"

	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

var (
	sessionID        = node_session.ID("SessionID")
	ProviderID       = identity.FromAddress("ProviderID")
	ServiceType      = "ServiceType"
	ProviderCountry  = "ProviderCountry"
	SessionStatsMock = consumer.SessionStatistics{
		BytesReceived: 10,
		BytesSent:     10,
	}
	SessionMock = session.Session{
		SessionID:       sessionID,
		ProviderID:      ProviderID,
		ServiceType:     ServiceType,
		ProviderCountry: ProviderCountry,
		Started:         time.Now(),
		Updated:         time.Now(),
		DataStats:       SessionStatsMock,
	}
)

func TestSessionToDto(t *testing.T) {
	value := "2010-01-01 12:00:00"
	format := "2006-01-02 15:04:05"
	startedAt, _ := time.Parse(format, value)

	se := SessionMock
	se.Started = startedAt
	sessionDTO := sessionToDto(se)

	assert.Equal(t, value, sessionDTO.DateStarted)
	assert.Equal(t, string(SessionMock.SessionID), sessionDTO.SessionID)
	assert.Equal(t, SessionMock.ProviderID.Address, sessionDTO.ProviderID)
	assert.Equal(t, SessionMock.ServiceType, sessionDTO.ServiceType)
	assert.Equal(t, SessionMock.ProviderCountry, sessionDTO.ProviderCountry)
	assert.Equal(t, SessionMock.DataStats.BytesReceived, sessionDTO.BytesReceived)
	assert.Equal(t, SessionMock.DataStats.BytesSent, sessionDTO.BytesSent)
	assert.Equal(t, SessionMock.GetDuration(), sessionDTO.Duration)
	assert.Equal(t, SessionMock.Status.String(), sessionDTO.Status)
}
