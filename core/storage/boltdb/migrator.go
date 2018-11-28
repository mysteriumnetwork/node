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

package boltdb

import (
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations"
)

const migrationLogPrefix = "[migrator] "
const migrationIndexBucketName = "migrations"

// Migrator represents the component responsible for running migrations on bolt db
type Migrator struct {
	db         *Bolt
	migrations []migrations.Migration
}

// NewMigrator returns a new instance of migrator
func NewMigrator(db *Bolt) *Migrator {
	return &Migrator{
		db: db,
		migrations: []migrations.Migration{
			migrations.Migration{
				Name:    "session-to-session-history",
				Migrate: migrations.MigrateSessionToHistory,
			},
		},
	}
}

func (m *Migrator) isMigrationRun(migration migrations.Migration) (bool, error) {
	migrations := []migrations.Migration{}
	err := m.db.db.All(&migrations)
	if err != nil {
		return true, err
	}

	for i := range migrations {
		if migration.Name == migrations[i].Name {
			return true, nil
		}
	}
	return false, nil
}

func (m *Migrator) saveMigrationRun(migration migrations.Migration) error {
	return m.db.db.Save(&migration)
}

func (m *Migrator) migrate(migration migrations.Migration) error {
	isRun, err := m.isMigrationRun(migration)
	if err != nil {
		return err
	}
	if isRun {
		return nil
	}
	log.Info(migrationLogPrefix, "running migration", migration.Name)
	err = migration.Migrate(m.db.db)
	if err != nil {
		return err
	}
	log.Info(migrationLogPrefix, "saving migration", migration.Name)
	err = m.saveMigrationRun(migration)
	return err
}

// Up updates the bolt DB to the latest version
func (m *Migrator) Up() error {
	for i := range m.migrations {
		err := m.migrate(m.migrations[i])
		if err != nil {
			return err
		}
	}
	return nil
}
