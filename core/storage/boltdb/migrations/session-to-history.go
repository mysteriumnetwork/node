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
	"time"

	"github.com/asdine/storm"
	"github.com/mysteriumnetwork/node/consumer"
	consumer_session "github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
	"github.com/rs/zerolog/log"
)

// Status represents list of possible session statuses
type Status int

// Session holds structure for saving session history
type Session struct {
	SessionID       node_session.ID `storm:"id"`
	ProviderID      identity.Identity
	ServiceType     string
	ProviderCountry string
	Started         time.Time
	Status          Status
	Updated         time.Time
	DataStats       consumer.SessionStatistics // is updated on disconnect event
}

// ToSessionHistory converts the session struct to a session history struct
func (s Session) ToSessionHistory() consumer_session.History {
	status := ""
	if s.Status == 0 {
		status = "New"
	} else if s.Status == 1 {
		status = "Completed"
	}

	return consumer_session.History{
		SessionID:       s.SessionID,
		ProviderID:      s.ProviderID,
		ServiceType:     s.ServiceType,
		ProviderCountry: s.ProviderCountry,
		Started:         s.Started,
		Status:          status,
		Updated:         s.Updated,
		DataStats:       s.DataStats,
	}
}

// MigrateSessionToHistory runs the session to session history migration
func MigrateSessionToHistory(db *storm.DB) error {
	res := []Session{}
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}

	err = tx.All(&res)
	if err != nil {
		return err
	}

	historyBucket := tx.From("session-history")
	for i := range res {
		sh := res[i].ToSessionHistory()
		err := historyBucket.Save(&sh)
		if err != nil {
			rollbackError := tx.Rollback()
			if rollbackError != nil {
				log.Error().Err(err).Stack().Msg("Migrate session to history rollback failed!")
			}
			return err
		}
	}
	return tx.Commit()
}
