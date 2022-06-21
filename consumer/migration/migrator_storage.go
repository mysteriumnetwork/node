/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package migration

import (
	"errors"
	"fmt"

	storm "github.com/asdine/storm/v3"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/rs/zerolog/log"
)

const hermesMigrationBucketName = "hermes_migration"
const hermesMigrationFinishedKey = "migration_finished"

// Storage keeps track of migration progress
type Storage struct {
	db respository
	ap registry.AddressProvider
}

type respository interface {
	SetValue(bucket string, key interface{}, to interface{}) error
	GetValue(bucket string, key interface{}, to interface{}) error
}

// NewStorage builds and returns a new storage object.
func NewStorage(db respository, ap registry.AddressProvider) *Storage {
	return &Storage{
		db: db,
		ap: ap,
	}
}

// MarkAsMigrated set migration flag do not try to migrate again
func (s *Storage) MarkAsMigrated(chainID int64, identity string) {
	activeHermes, err := s.ap.GetActiveHermes(chainID)
	if err != nil {
		log.Err(err).Msg("No active hermes set, will skip marking migration status")
		return
	}

	err = s.db.SetValue(hermesMigrationBucketName, s.getMigrationKey(activeHermes.Hex(), identity), true)
	if err != nil {
		log.Warn().Err(err).Msg("Could not save migration state to local db")
	}
}

func (s *Storage) isMigrationRequired(chainID int64, identity string) bool {
	activeHermes, err := s.ap.GetActiveHermes(chainID)
	if err != nil {
		log.Err(err).Msg("No active hermes set, will assume required")
		return true
	}

	var finished bool
	err = s.db.GetValue(hermesMigrationBucketName, s.getMigrationKey(activeHermes.Hex(), identity), &finished)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			log.Warn().Err(err).Msg("Could not get migration state from local db")
		}
		return true
	}

	return !finished
}

func (s *Storage) getMigrationKey(hermesId, id string) string {
	return fmt.Sprintf("%s_%s_%s", hermesMigrationFinishedKey, hermesId, id)
}
